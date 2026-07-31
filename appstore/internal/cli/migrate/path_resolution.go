package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
)

type pathSource string

const (
	pathSourceFlag        pathSource = "flag"
	pathSourceDeliverfile pathSource = "deliverfile"
	pathSourceDefault     pathSource = "default"
)

type importInputOptions struct {
	WorkDir         string
	FastlaneDir     string
	MetadataDir     string
	ScreenshotsDir  string
	SkipScreenshots bool
	// The allow options preserve legacy Fastlane layouts only when the operator
	// explicitly trusts paths that repository-controlled configuration can
	// redirect outside the selected checkout.
	AllowExternalMetadata     bool
	AllowExternalScreenshots  bool
	AllowSymlinkedDeliverfile bool
}

type importInputs struct {
	DeliverfilePath   string
	DeliverfileConfig DeliverfileConfig
	MetadataDir       string
	ScreenshotsDir    string
	MetadataSource    pathSource
	ScreenshotsSource pathSource
}

func resolveImportInputs(opts importInputOptions) (importInputs, []SkippedItem, error) {
	workDir := opts.WorkDir
	if strings.TrimSpace(workDir) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return importInputs{}, nil, fmt.Errorf("resolve import paths: %w", err)
		}
		workDir = cwd
	}

	if opts.FastlaneDir != "" {
		if err := ensureDirExists(opts.FastlaneDir); err != nil {
			return importInputs{}, nil, err
		}
	}

	deliverfilePath, err := discoverDeliverfilePath(workDir, opts.FastlaneDir)
	if err != nil {
		return importInputs{}, nil, err
	}

	var config DeliverfileConfig
	if deliverfilePath != "" {
		config, err = readDeliverfileConfig(workDir, opts.FastlaneDir, deliverfilePath, opts.AllowSymlinkedDeliverfile)
		if err != nil {
			return importInputs{}, nil, err
		}
	}

	inputs := importInputs{
		DeliverfilePath:   deliverfilePath,
		DeliverfileConfig: config,
	}

	metadataDir, metadataSource, err := resolveImportPath(workDir, opts.FastlaneDir, deliverfilePath, opts.MetadataDir, config.MetadataPath, "metadata", opts.AllowExternalMetadata)
	if err != nil {
		return importInputs{}, nil, err
	}
	screenshotsDir, screenshotsSource, err := resolveImportPath(workDir, opts.FastlaneDir, deliverfilePath, opts.ScreenshotsDir, config.ScreenshotsPath, "screenshots", opts.AllowExternalScreenshots)
	if err != nil {
		return importInputs{}, nil, err
	}
	skipScreenshots := opts.SkipScreenshots || config.SkipScreenshots

	skipped := []SkippedItem{}
	metadataDir, skipped, err = validateResolvedDir(metadataDir, metadataSource, "metadata", skipped)
	if err != nil {
		return importInputs{}, nil, err
	}
	if !skipScreenshots {
		screenshotsDir, skipped, err = validateResolvedDir(screenshotsDir, screenshotsSource, "screenshots", skipped)
		if err != nil {
			return importInputs{}, nil, err
		}
	}

	inputs.MetadataDir = metadataDir
	inputs.ScreenshotsDir = screenshotsDir
	inputs.MetadataSource = metadataSource
	inputs.ScreenshotsSource = screenshotsSource

	return inputs, skipped, nil
}

func resolveImportPath(workDir, fastlaneDir, deliverfilePath, explicitPath, deliverfilePathValue, defaultDir string, allowExternal bool) (string, pathSource, error) {
	if strings.TrimSpace(explicitPath) != "" {
		return explicitPath, pathSourceFlag, nil
	}
	if strings.TrimSpace(fastlaneDir) != "" {
		resolved, err := resolveFastlaneChild(fastlaneDir, defaultDir, allowExternal)
		if err != nil {
			return "", pathSourceFlag, err
		}
		return resolved, pathSourceFlag, nil
	}
	if strings.TrimSpace(deliverfilePathValue) != "" {
		base := workDir
		if deliverfilePath != "" {
			base = filepath.Dir(deliverfilePath)
		}
		resolved, err := containDeliverfilePath(workDir, base, deliverfilePathValue)
		if err != nil {
			if !allowExternal {
				return "", pathSourceDeliverfile, err
			}
			resolved, err = resolveTrustedDeliverfilePath(base, deliverfilePathValue)
			if err != nil {
				return "", pathSourceDeliverfile, err
			}
		}
		return resolved, pathSourceDeliverfile, nil
	}
	base := workDir
	if deliverfilePath != "" {
		base = filepath.Dir(deliverfilePath)
	}
	return filepath.Join(base, defaultDir), pathSourceDefault, nil
}

func resolveFastlaneChild(fastlaneDir, child string, allowExternal bool) (string, error) {
	resolved, err := containFastlaneChild(fastlaneDir, child)
	if err == nil {
		return resolved, nil
	}
	if !allowExternal {
		return "", err
	}
	return filepath.EvalSymlinks(filepath.Join(fastlaneDir, child))
}

func resolveTrustedDeliverfilePath(base, value string) (string, error) {
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

// containFastlaneChild resolves a conventional child directory beneath the
// operator-selected fastlane root and refuses a symlinked child, because the
// fastlane checkout's contents are repository-controlled even when the root
// itself was chosen by the operator.
func containFastlaneChild(fastlaneDir, child string) (string, error) {
	root, err := rootfs.New(fastlaneDir)
	if err != nil {
		return "", err
	}
	if err := root.CheckContained(child); err != nil {
		return "", err
	}
	return filepath.Join(fastlaneDir, child), nil
}

// containDeliverfilePath resolves a repository-controlled Deliverfile path
// value against the Deliverfile's own directory and requires the result to stay
// inside the working directory, because the Deliverfile ships with the checkout
// and must not select files outside it.
func containDeliverfilePath(workDir, base, value string) (string, error) {
	if err := rootfs.ValidateRelativeAllowingTraversal(value); err != nil {
		return "", fmt.Errorf("deliverfile path %q: %w", value, err)
	}
	root, err := rootfs.New(workDir)
	if err != nil {
		return "", err
	}
	// The Deliverfile directory may itself be relative to the process working
	// directory; make it absolute so joining below cannot re-resolve it against
	// the trusted root and duplicate the leading components.
	absoluteBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	resolved, err := root.Resolve(filepath.Clean(filepath.Join(absoluteBase, value)))
	if err != nil {
		return "", fmt.Errorf("deliverfile path %q must stay inside %s: %w", value, root.Path(), err)
	}
	return resolved, nil
}

func validateResolvedDir(path string, source pathSource, label string, skipped []SkippedItem) (string, []SkippedItem, error) {
	if strings.TrimSpace(path) == "" {
		return "", skipped, nil
	}
	if err := ensureDirExists(path); err != nil {
		if source == pathSourceDefault {
			skipped = append(skipped, SkippedItem{
				Path:   path,
				Reason: fmt.Sprintf("default %s directory not found", label),
			})
			return "", skipped, nil
		}
		return "", skipped, err
	}
	return path, skipped, nil
}

func discoverDeliverfilePath(workDir, fastlaneDir string) (string, error) {
	if strings.TrimSpace(fastlaneDir) != "" {
		path := filepath.Join(fastlaneDir, "Deliverfile")
		if exists, err := fileExists(path); err != nil {
			return "", err
		} else if exists {
			return path, nil
		}
		return "", nil
	}

	candidates := []string{
		filepath.Join(workDir, "Deliverfile"),
		filepath.Join(workDir, "fastlane", "Deliverfile"),
	}
	for _, path := range candidates {
		if exists, err := fileExists(path); err != nil {
			return "", err
		} else if exists {
			return path, nil
		}
	}
	return "", nil
}

// readDeliverfileConfig reads repository-controlled Deliverfiles through a
// rooted no-follow handle. The legacy symlink-following behavior remains
// available only behind an explicit trust decision.
func readDeliverfileConfig(workDir, fastlaneDir, path string, allowSymlink bool) (DeliverfileConfig, error) {
	if allowSymlink {
		return parseDeliverfile(path)
	}
	rootPath := workDir
	if strings.TrimSpace(fastlaneDir) != "" {
		rootPath = fastlaneDir
	}
	root, err := rootfs.New(rootPath)
	if err != nil {
		return DeliverfileConfig{}, err
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return DeliverfileConfig{}, err
	}
	relative, err := filepath.Rel(root.Path(), absolutePath)
	if err != nil {
		return DeliverfileConfig{}, err
	}
	file, err := root.OpenFile(relative)
	if err != nil {
		return DeliverfileConfig{}, fmt.Errorf("read Deliverfile: %w", err)
	}
	defer file.Close()
	return parseDeliverfileReader(path, file)
}

func ensureDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}
	return nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode().IsRegular(), nil
}
