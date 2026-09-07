package notarization

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	// errStaplerRegularFileDirectory tells the caller that the no-follow
	// regular-file walker found a directory. Directory bundles continue through
	// the existing rooted directory path, which has the read semantics required
	// for inventory capture.
	errStaplerRegularFileDirectory = errors.New("stapler target is a directory")
	// errStaplerRegularFileUnsupported is returned on platforms where this
	// private search-only capability is unavailable. The existing rooted path is
	// retained there rather than weakening its guarantees.
	errStaplerRegularFileUnsupported        = errors.New("search-only regular-file traversal is unsupported")
	errStaplerRegularFileParentNotDirectory = errors.New("regular-file parent is not a directory")
)

type staplerRegularFileNotRegularError struct {
	info os.FileInfo
}

func (e *staplerRegularFileNotRegularError) Error() string {
	return "stapler target is not a regular file"
}

func (e *staplerRegularFileNotRegularError) Unwrap() error {
	return errStaplerRegularFileNotRegular
}

var errStaplerRegularFileNotRegular = errors.New("stapler target is not regular")

type staplerRegularFileOpenOps struct {
	openSearchAt func(parent *os.File, name string) (*os.File, error)
	openFinalAt  func(parent *os.File, name string) (*os.File, error)
}

type staplerRegularFileStep struct {
	name string
	dir  *os.File
	info os.FileInfo
}

type staplerRegularFileWalker struct {
	ops         staplerRegularFileOpenOps
	anchor      *os.File
	anchorOwned bool
	anchorInfo  os.FileInfo
	steps       []staplerRegularFileStep
	finalParent *os.File
	finalName   string
	final       *os.File
	identity    os.FileInfo
	ownedDirs   []*os.File
}

// staplerRegularFileAccess is a private capability for a regular artifact
// whose parent directories cannot be read but can be searched. It retains the
// anchor, every walked parent, and the initially opened final descriptor. All
// later opens/probes re-walk the retained path with no-follow openat calls.
type staplerRegularFileAccess struct {
	final        *os.File
	identity     os.FileInfo
	openFn       func() (*os.File, error)
	verifyPathFn func() (os.FileInfo, error)
	closeFn      func() error
}

func (access *staplerRegularFileAccess) open() (*os.File, error) {
	if access == nil || access.openFn == nil {
		return nil, errors.New("regular-file access capability is missing")
	}
	return access.openFn()
}

func (access *staplerRegularFileAccess) probe() (os.FileInfo, error) {
	if access == nil || access.verifyPathFn == nil {
		return nil, errors.New("regular-file access capability is missing")
	}
	return access.verifyPathFn()
}

func (access *staplerRegularFileAccess) close() error {
	if access == nil || access.closeFn == nil {
		return nil
	}
	return access.closeFn()
}

func (walker *staplerRegularFileWalker) close() error {
	if walker == nil {
		return nil
	}
	var closeErr error
	if walker.final != nil {
		closeErr = errors.Join(closeErr, walker.final.Close())
		walker.final = nil
	}
	for index := len(walker.ownedDirs) - 1; index >= 0; index-- {
		closeErr = errors.Join(closeErr, walker.ownedDirs[index].Close())
	}
	walker.ownedDirs = nil
	if walker.anchorOwned && walker.anchor != nil {
		closeErr = errors.Join(closeErr, walker.anchor.Close())
		walker.anchor = nil
	}
	return closeErr
}

func (walker *staplerRegularFileWalker) openFinal() (*os.File, error) {
	if _, err := walker.verifyPath(); err != nil {
		return nil, err
	}
	opened, err := walker.ops.openFinalAt(walker.finalParent, walker.finalName)
	if err != nil {
		return nil, err
	}
	info, err := opened.Stat()
	if err != nil {
		_ = opened.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(walker.identity, info) {
		_ = opened.Close()
		return nil, errStaplerTargetRaced
	}
	return opened, nil
}

// verifyPath re-walks from the retained anchor so a replacement of any
// parent component is detected. It deliberately opens temporary descriptors
// for the current pathname instead of trusting the original parent handles.
func (walker *staplerRegularFileWalker) verifyPath() (os.FileInfo, error) {
	if walker == nil || walker.anchor == nil || walker.finalParent == nil {
		return nil, errors.New("regular-file path capability is incomplete")
	}
	anchorInfo, err := walker.anchor.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(walker.anchorInfo, anchorInfo) || !anchorInfo.IsDir() {
		return nil, errStaplerTargetRaced
	}

	current := walker.anchor
	var temporary []*os.File
	defer func() {
		for index := len(temporary) - 1; index >= 0; index-- {
			_ = temporary[index].Close()
		}
	}()
	for _, step := range walker.steps {
		next, openErr := walker.ops.openSearchAt(current, step.name)
		if openErr != nil {
			return nil, openErr
		}
		info, statErr := next.Stat()
		if statErr != nil {
			_ = next.Close()
			return nil, statErr
		}
		if !info.IsDir() || !os.SameFile(step.info, info) {
			_ = next.Close()
			return nil, errStaplerTargetRaced
		}
		temporary = append(temporary, next)
		current = next
	}

	final, openErr := walker.ops.openFinalAt(current, walker.finalName)
	if openErr != nil {
		return nil, openErr
	}
	info, statErr := final.Stat()
	closeErr := final.Close()
	if statErr != nil {
		return nil, statErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if !info.Mode().IsRegular() || !os.SameFile(walker.identity, info) {
		return nil, errStaplerTargetRaced
	}
	return info, nil
}

func staplerRegularFileTargetWithin(anchorPath, targetPath string) (string, bool) {
	relative, err := filepath.Rel(anchorPath, targetPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	if relative == "." || relative == "" {
		return "", false
	}
	return relative, true
}

func newStaplerRegularFileAccessWithOps(absolute, workingDirectoryPath string, workingDirectory *os.File, ops staplerRegularFileOpenOps) (*staplerRegularFileAccess, error) {
	canonical := filepath.Clean(staplerNoFollowPath(absolute))
	volumeRoot := filepath.VolumeName(canonical) + string(filepath.Separator)
	if volumeRoot == string(filepath.Separator) && filepath.VolumeName(canonical) != "" {
		volumeRoot = filepath.VolumeName(canonical) + string(filepath.Separator)
	}

	walker := &staplerRegularFileWalker{
		ops: ops,
	}
	var relative string
	if workingDirectory != nil && workingDirectoryPath != "" {
		anchorPath := filepath.Clean(staplerNoFollowPath(workingDirectoryPath))
		if candidate, ok := staplerRegularFileTargetWithin(anchorPath, canonical); ok {
			walker.anchor = workingDirectory
			walker.anchorInfo, _ = workingDirectory.Stat()
			if walker.anchorInfo == nil || !walker.anchorInfo.IsDir() {
				return nil, errors.New("retained current working directory is not searchable")
			}
			relative = candidate
		}
	}
	if walker.anchor == nil {
		anchor, err := openStaplerSearchableDirectoryNoFollow(volumeRoot)
		if err != nil {
			return nil, errStaplerRegularFileUnsupported
		}
		walker.anchor = anchor
		walker.anchorOwned = true
		walker.anchorInfo, err = anchor.Stat()
		if err != nil {
			_ = walker.close()
			return nil, err
		}
		relative, _ = staplerRegularFileTargetWithin(volumeRoot, canonical)
		if relative == "" {
			_ = walker.close()
			return nil, os.ErrInvalid
		}
	}

	parts := strings.FieldsFunc(relative, isStaplerPathSeparator)
	if len(parts) == 0 {
		_ = walker.close()
		return nil, os.ErrInvalid
	}
	current := walker.anchor
	for _, part := range parts[:len(parts)-1] {
		next, err := ops.openSearchAt(current, part)
		if err != nil {
			_ = walker.close()
			return nil, err
		}
		info, statErr := next.Stat()
		if statErr != nil {
			_ = next.Close()
			_ = walker.close()
			return nil, statErr
		}
		if !info.IsDir() {
			_ = next.Close()
			_ = walker.close()
			return nil, errStaplerRegularFileParentNotDirectory
		}
		walker.steps = append(walker.steps, staplerRegularFileStep{name: part, dir: next, info: info})
		walker.ownedDirs = append(walker.ownedDirs, next)
		current = next
	}
	walker.finalParent = current
	walker.finalName = parts[len(parts)-1]
	final, err := ops.openFinalAt(current, walker.finalName)
	if err != nil {
		_ = walker.close()
		return nil, err
	}
	info, err := final.Stat()
	if err != nil {
		_ = final.Close()
		_ = walker.close()
		return nil, err
	}
	if info.IsDir() {
		_ = final.Close()
		_ = walker.close()
		return nil, errStaplerRegularFileDirectory
	}
	if !info.Mode().IsRegular() {
		_ = final.Close()
		_ = walker.close()
		return nil, &staplerRegularFileNotRegularError{info: info}
	}
	walker.final = final
	walker.identity = info
	return &staplerRegularFileAccess{
		final:        final,
		identity:     info,
		openFn:       walker.openFinal,
		verifyPathFn: walker.verifyPath,
		closeFn:      walker.close,
	}, nil
}
