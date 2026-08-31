package distribution

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

const maxMaterializedAppExpandedBytes int64 = 4 << 30

// MaterializedApp is the verified main app extracted into a private temporary
// directory. The path is valid until Cleanup is called.
type MaterializedApp struct {
	Inspection Inspection
	Path       string

	cleanup func()
}

// MaterializationError reports a failure after IPA inspection completed. The
// inspection remains available so callers can return artifact identity in a
// structured failure without retaining any source path or archive contents.
type MaterializationError struct {
	Inspection Inspection
	Err        error
}

func (e *MaterializationError) Error() string {
	if e == nil || e.Err == nil {
		return "IPA materialization failed"
	}
	return e.Err.Error()
}

func (e *MaterializationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func materializationError(inspection Inspection, err error) error {
	if err == nil {
		return nil
	}
	return &MaterializationError{Inspection: inspection, Err: err}
}

// Cleanup removes the temporary app directory. It is safe to call more than
// once, although callers should normally defer it immediately after success.
func (app *MaterializedApp) Cleanup() {
	if app == nil || app.cleanup == nil {
		return
	}
	cleanup := app.cleanup
	app.cleanup = nil
	cleanup()
}

// IPAAppSource preserves one immutable inspected IPA snapshot so the verified
// main app can be extracted later from the exact byte generation that was
// inspected. Opening a source never extracts app payload bytes; extraction
// happens only when MaterializeApp is called. The caller must call Cleanup.
type IPAAppSource struct {
	inspection      Inspection
	snapshot        *os.File
	size            int64
	cleanupSnapshot func()
}

// OpenIPAAppSourceContext snapshots and inspects an already-open IPA without
// extracting any app payload bytes, so callers can run cheap eligibility
// checks against Inspection before paying for extraction with MaterializeApp.
func OpenIPAAppSourceContext(ctx context.Context, source *os.File, size int64, options InspectOptions) (*IPAAppSource, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot, digest, cleanupSnapshot, err := snapshotIPAContext(ctx, source, size)
	if err != nil {
		return nil, err
	}
	inspection, err := inspectSnapshotContext(ctx, snapshot, size, digest, options)
	if err != nil {
		cleanupSnapshot()
		return nil, fmt.Errorf("IPA preflight: %w", err)
	}
	return &IPAAppSource{
		inspection:      inspection,
		snapshot:        snapshot,
		size:            size,
		cleanupSnapshot: cleanupSnapshot,
	}, nil
}

// Inspection returns the inspection captured from the preserved snapshot.
func (s *IPAAppSource) Inspection() Inspection {
	if s == nil {
		return Inspection{}
	}
	return s.inspection
}

// MaterializeApp extracts only the single inspected Payload/*.app subtree from
// the preserved snapshot into a fresh private temporary directory. The
// snapshot used for Inspection and extraction is the same file, so the
// returned app cannot be assembled from different source byte generations.
// The caller owns the returned app and must call its Cleanup in addition to
// the source Cleanup.
func (s *IPAAppSource) MaterializeApp(ctx context.Context) (*MaterializedApp, error) {
	if s == nil || s.snapshot == nil {
		return nil, fmt.Errorf("IPA app source is closed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	appRoot, err := os.MkdirTemp("", ".asc-xcode-install-app-")
	if err != nil {
		return nil, materializationError(s.inspection, fmt.Errorf("create private app directory: %w", err))
	}
	if err := os.Chmod(appRoot, 0o700); err != nil {
		_ = os.RemoveAll(appRoot)
		return nil, materializationError(s.inspection, fmt.Errorf("protect private app directory: %w", err))
	}
	root, err := os.OpenRoot(appRoot)
	if err != nil {
		_ = os.RemoveAll(appRoot)
		return nil, materializationError(s.inspection, fmt.Errorf("open private app directory: %w", err))
	}
	appName, err := materializeMainAppFromSnapshot(ctx, root, s.snapshot, s.size)
	if err != nil {
		_ = root.Close()
		_ = os.RemoveAll(appRoot)
		return nil, materializationError(s.inspection, fmt.Errorf("materialize IPA app: %w", err))
	}
	if err := root.Close(); err != nil {
		_ = os.RemoveAll(appRoot)
		return nil, materializationError(s.inspection, fmt.Errorf("close private app directory: %w", err))
	}
	return &MaterializedApp{
		Inspection: s.inspection,
		Path:       filepath.Join(appRoot, filepath.FromSlash(appName)),
		cleanup:    func() { _ = os.RemoveAll(appRoot) },
	}, nil
}

// Cleanup releases the preserved snapshot. It is safe to call more than once.
func (s *IPAAppSource) Cleanup() {
	if s == nil || s.cleanupSnapshot == nil {
		return
	}
	cleanup := s.cleanupSnapshot
	s.cleanupSnapshot = nil
	s.snapshot = nil
	cleanup()
}

func materializeMainAppFromSnapshot(ctx context.Context, destination *os.Root, file *os.File, size int64) (string, error) {
	reader, err := zip.NewReader(file, size)
	if err != nil {
		return "", fmt.Errorf("open IPA for app materialization: %w", err)
	}
	appDir, err := findMainAppDirectory(reader.File)
	if err != nil {
		return "", err
	}
	appName := path.Base(appDir)
	if err := destination.MkdirAll(appName, 0o700); err != nil {
		return "", fmt.Errorf("create materialized app directory: %w", err)
	}
	prefix := appDir + "/"
	var total int64
	for _, member := range reader.File {
		if err := contextError(ctx); err != nil {
			return "", err
		}
		if !strings.HasPrefix(member.Name, prefix) {
			continue
		}
		if err := validateArchiveMember(member); err != nil {
			return "", err
		}
		relative := strings.TrimPrefix(member.Name, prefix)
		if relative == "" {
			continue
		}
		target := filepath.Join(filepath.FromSlash(appName), filepath.FromSlash(relative))
		if member.FileInfo().IsDir() {
			if err := destination.MkdirAll(strings.TrimSuffix(target, string(filepath.Separator)), 0o700); err != nil {
				return "", fmt.Errorf("create materialized app directory: %w", err)
			}
			continue
		}
		if member.UncompressedSize64 > uint64(maxMaterializedAppExpandedBytes) || total > maxMaterializedAppExpandedBytes-int64(member.UncompressedSize64) {
			return "", fmt.Errorf("expanded main app exceeds %d bytes", maxMaterializedAppExpandedBytes)
		}
		total += int64(member.UncompressedSize64)
		if err := destination.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return "", fmt.Errorf("create materialized app parent directory: %w", err)
		}
		mode := member.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		if err := copyZipMemberToNewFileContextWithMode(ctx, destination, target, member, int64(member.UncompressedSize64), mode); err != nil {
			return "", fmt.Errorf("materialize app member %q: %w", relative, err)
		}
	}
	return appName, nil
}

func findMainAppDirectory(members []*zip.File) (string, error) {
	var appDir string
	for _, member := range members {
		if !isMainAppMember(member.Name, "Info.plist") || member.FileInfo().IsDir() {
			continue
		}
		if appDir != "" {
			return "", fmt.Errorf("IPA contains multiple main apps")
		}
		appDir = path.Dir(member.Name)
	}
	if appDir == "" {
		return "", fmt.Errorf("IPA is missing Payload/*.app/Info.plist")
	}
	return appDir, nil
}

func copyZipMemberToNewFileContextWithMode(ctx context.Context, root *os.Root, name string, member *zip.File, limit int64, mode os.FileMode) error {
	reader, err := member.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	file, err := secureopen.OpenNewFileNoFollowInRoot(root, name, mode)
	if err != nil {
		return err
	}
	written, copyErr := copyWithContext(ctx, file, io.LimitReader(reader, limit+1), nil)
	chmodErr := file.Chmod(mode)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if chmodErr != nil {
		return chmodErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > limit {
		return fmt.Errorf("expanded member exceeds %d bytes", limit)
	}
	if written != limit {
		return fmt.Errorf("expanded member size does not match its declaration")
	}
	return nil
}
