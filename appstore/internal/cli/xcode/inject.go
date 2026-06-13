package xcode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"howett.net/plist"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

type xcodeInjectManifest struct {
	Values  map[string]any              `json:"values,omitempty"`
	Outputs []xcodeInjectManifestOutput `json:"outputs"`
}

type xcodeInjectManifestOutput struct {
	Type     string         `json:"type"`
	Path     string         `json:"path"`
	Source   string         `json:"source,omitempty"`
	Values   map[string]any `json:"values,omitempty"`
	Contents string         `json:"contents,omitempty"`
}

type xcodeInjectOptions struct {
	ManifestPath string
	SetValues    []string
	Overwrite    bool
	DryRun       bool
}

type xcodeInjectResult struct {
	ManifestPath string                  `json:"manifest_path"`
	DryRun       bool                    `json:"dry_run"`
	Outputs      []xcodeInjectFileResult `json:"outputs"`
}

type xcodeInjectFileResult struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	Action string `json:"action"`
	Bytes  int64  `json:"bytes,omitempty"`
}

type xcodeInjectOutputPlan struct {
	Type    string
	Path    string
	Source  string
	Payload []byte
}

func XcodeInjectCommand() *ffcli.Command {
	fs := flag.NewFlagSet("xcode inject", flag.ExitOnError)

	manifestPath := fs.String("manifest", "", "Path to deployment metadata manifest JSON (required)")
	overwrite := fs.Bool("overwrite", false, "Replace existing output files")
	dryRun := fs.Bool("dry-run", false, "Validate and print the files that would be generated without writing")
	var setValues shared.MultiStringFlag
	fs.Var(&setValues, "set", "Override a manifest value as key=value (repeatable)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "inject",
		ShortUsage: "asc xcode inject --manifest FILE [flags]",
		ShortHelp:  "[experimental] Generate Xcode deployment metadata files from a manifest.",
		LongHelp: `[experimental] Generate Xcode deployment metadata files from a manifest.

The manifest is JSON with top-level "values" and "outputs" fields. Outputs can
generate plist, json, or text files, and can copy declared assets such as app
icons into asset catalog paths. Relative output paths and copy sources resolve
from the manifest directory.

String values support ${key} placeholders from manifest values and repeated
--set key=value overrides.

Example manifest:
  {
    "values": {
      "bundle_id": "com.example.app",
      "app_name": "Example",
      "version": "1.2.3",
      "build_number": "42"
    },
    "outputs": [
      {
        "type": "plist",
        "path": "Generated/Info.generated.plist",
        "values": {
          "CFBundleIdentifier": "${bundle_id}",
          "CFBundleDisplayName": "${app_name}",
          "CFBundleShortVersionString": "${version}",
          "CFBundleVersion": "${build_number}"
        }
      },
      {
        "type": "copy",
        "source": "Assets/AppIcon.appiconset/Contents.json",
        "path": "App/Assets.xcassets/AppIcon.appiconset/Contents.json"
      }
    ]
  }

Examples:
  asc xcode inject --manifest .asc/deployment.json --dry-run --output json
  asc xcode inject --manifest .asc/deployment.json --set version=1.2.4 --set build_number=43 --overwrite`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}
			result, err := runXcodeInject(xcodeInjectOptions{
				ManifestPath: strings.TrimSpace(*manifestPath),
				SetValues:    []string(setValues),
				Overwrite:    *overwrite,
				DryRun:       *dryRun,
			})
			if err != nil {
				var usageErr xcodeInjectUsageError
				if errors.As(err, &usageErr) {
					return shared.UsageError(usageErr.Error())
				}
				return fmt.Errorf("xcode inject: %w", err)
			}

			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error {
					asc.RenderTable([]string{"type", "action", "path", "source"}, xcodeInjectResultRows(result))
					return nil
				},
				func() error {
					asc.RenderMarkdown([]string{"type", "action", "path", "source"}, xcodeInjectResultRows(result))
					return nil
				},
			)
		},
	}
}

type xcodeInjectUsageError struct {
	message string
}

func (e xcodeInjectUsageError) Error() string {
	return e.message
}

func newXcodeInjectUsageError(format string, args ...any) error {
	return xcodeInjectUsageError{message: fmt.Sprintf(format, args...)}
}

func runXcodeInject(opts xcodeInjectOptions) (xcodeInjectResult, error) {
	opts.ManifestPath = strings.TrimSpace(opts.ManifestPath)
	if opts.ManifestPath == "" {
		return xcodeInjectResult{}, newXcodeInjectUsageError("--manifest is required")
	}

	manifest, err := readXcodeInjectManifest(opts.ManifestPath)
	if err != nil {
		return xcodeInjectResult{}, err
	}
	if len(manifest.Outputs) == 0 {
		return xcodeInjectResult{}, newXcodeInjectUsageError("manifest outputs must include at least one entry")
	}

	values := make(map[string]any, len(manifest.Values)+len(opts.SetValues))
	for key, value := range manifest.Values {
		values[key] = value
	}
	if err := applyXcodeInjectSetValues(values, opts.SetValues); err != nil {
		return xcodeInjectResult{}, err
	}

	baseDir := filepath.Dir(opts.ManifestPath)
	if err := validateXcodeInjectOutputDestinations(baseDir, manifest.Outputs, opts.Overwrite); err != nil {
		return xcodeInjectResult{}, err
	}

	result := xcodeInjectResult{
		ManifestPath: opts.ManifestPath,
		DryRun:       opts.DryRun,
		Outputs:      make([]xcodeInjectFileResult, 0, len(manifest.Outputs)),
	}
	plans := make([]xcodeInjectOutputPlan, 0, len(manifest.Outputs))

	for i, output := range manifest.Outputs {
		plan, err := planXcodeInjectOutput(baseDir, values, output)
		if err != nil {
			return xcodeInjectResult{}, fmt.Errorf("output %d: %w", i+1, err)
		}
		plans = append(plans, plan)
	}

	for i, plan := range plans {
		fileResult, err := writeXcodeInjectBytes(plan.Path, plan.Type, plan.Source, plan.Payload, opts)
		if err != nil {
			return xcodeInjectResult{}, fmt.Errorf("output %d: %w", i+1, err)
		}
		result.Outputs = append(result.Outputs, fileResult)
	}

	return result, nil
}

func validateXcodeInjectOutputDestinations(baseDir string, outputs []xcodeInjectManifestOutput, overwrite bool) error {
	seen := map[string]int{}
	paths := make([]string, 0, len(outputs))
	pathKeys := make([]string, 0, len(outputs))
	for i, output := range outputs {
		targetPath := resolveXcodeInjectPath(baseDir, strings.TrimSpace(output.Path))
		if targetPath == "" {
			continue
		}
		if err := validateXcodeInjectDestinationParents(targetPath); err != nil {
			return fmt.Errorf("output %d: %w", i+1, err)
		}
		if err := validateXcodeInjectDestination(targetPath, overwrite); err != nil {
			return fmt.Errorf("output %d: %w", i+1, err)
		}
		targetKey, err := xcodeInjectPathConflictKey(targetPath)
		if err != nil {
			return fmt.Errorf("output %d: %w", i+1, err)
		}
		if first, exists := seen[targetKey]; exists {
			return newXcodeInjectUsageError("duplicate output path %q in outputs %d and %d", targetPath, first+1, i+1)
		}
		seen[targetKey] = i
		for index, existingPath := range paths {
			existingKey := pathKeys[index]
			if xcodeInjectPathContains(existingKey, targetKey) {
				return newXcodeInjectUsageError("output path %q conflicts with nested output path %q", existingPath, targetPath)
			}
			if xcodeInjectPathContains(targetKey, existingKey) {
				return newXcodeInjectUsageError("output path %q conflicts with nested output path %q", targetPath, existingPath)
			}
		}
		paths = append(paths, targetPath)
		pathKeys = append(pathKeys, targetKey)
	}
	for i, output := range outputs {
		if strings.ToLower(strings.TrimSpace(output.Type)) != "copy" || strings.TrimSpace(output.Source) == "" {
			continue
		}
		sourcePath := resolveXcodeInjectPath(baseDir, strings.TrimSpace(output.Source))
		sourceKey, err := xcodeInjectPathConflictKey(sourcePath)
		if err != nil {
			return fmt.Errorf("output %d: %w", i+1, err)
		}
		if first, exists := seen[sourceKey]; exists {
			return newXcodeInjectUsageError("copy source %q in output %d is produced by output %d", sourcePath, i+1, first+1)
		}
	}
	return nil
}

func validateXcodeInjectDestinationParents(path string) error {
	parent := filepath.Dir(path)
	for {
		if parent == "." || parent == "" {
			return nil
		}
		info, err := os.Stat(parent)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("output parent path %q is not a directory", parent)
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		next := filepath.Dir(parent)
		if next == parent {
			return nil
		}
		parent = next
	}
}

func xcodeInjectPathConflictKey(path string) (string, error) {
	cleaned := filepath.Clean(path)
	current := filepath.Dir(cleaned)
	missing := []string{filepath.Base(cleaned)}
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			parts := append([]string{resolved}, missing...)
			return strings.ToLower(filepath.Clean(filepath.Join(parts...))), nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(current)
		missing = append([]string{filepath.Base(current)}, missing...)
		if parent == current {
			return strings.ToLower(cleaned), nil
		}
		current = parent
	}
}

func xcodeInjectPathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func readXcodeInjectManifest(path string) (xcodeInjectManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return xcodeInjectManifest{}, err
	}

	var manifest xcodeInjectManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return xcodeInjectManifest{}, newXcodeInjectUsageError("failed to parse --manifest: %v", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return xcodeInjectManifest{}, newXcodeInjectUsageError("failed to parse --manifest: multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return xcodeInjectManifest{}, newXcodeInjectUsageError("failed to parse --manifest: multiple JSON values")
	}
	return manifest, nil
}

func applyXcodeInjectSetValues(values map[string]any, setValues []string) error {
	for _, entry := range setValues {
		key, value, ok := strings.Cut(entry, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return newXcodeInjectUsageError("--set values must use key=value")
		}
		values[key] = strings.TrimSpace(value)
	}
	return nil
}

func planXcodeInjectOutput(baseDir string, values map[string]any, output xcodeInjectManifestOutput) (xcodeInjectOutputPlan, error) {
	outputType := strings.ToLower(strings.TrimSpace(output.Type))
	targetPath := resolveXcodeInjectPath(baseDir, strings.TrimSpace(output.Path))
	if targetPath == "" {
		return xcodeInjectOutputPlan{}, newXcodeInjectUsageError("path is required")
	}

	switch outputType {
	case "plist":
		renderedValues, err := renderXcodeInjectValue(output.Values, values)
		if err != nil {
			return xcodeInjectOutputPlan{}, err
		}
		payload, err := plist.Marshal(renderedValues, plist.XMLFormat)
		if err != nil {
			return xcodeInjectOutputPlan{}, err
		}
		return xcodeInjectOutputPlan{Type: outputType, Path: targetPath, Payload: payload}, nil
	case "json":
		renderedValues, err := renderXcodeInjectValue(output.Values, values)
		if err != nil {
			return xcodeInjectOutputPlan{}, err
		}
		payload, err := json.MarshalIndent(renderedValues, "", "  ")
		if err != nil {
			return xcodeInjectOutputPlan{}, err
		}
		payload = append(payload, '\n')
		return xcodeInjectOutputPlan{Type: outputType, Path: targetPath, Payload: payload}, nil
	case "text":
		renderedContents, err := renderXcodeInjectString(output.Contents, values)
		if err != nil {
			return xcodeInjectOutputPlan{}, err
		}
		return xcodeInjectOutputPlan{Type: outputType, Path: targetPath, Payload: []byte(renderedContents)}, nil
	case "copy":
		sourcePath := resolveXcodeInjectPath(baseDir, strings.TrimSpace(output.Source))
		if sourcePath == "" {
			return xcodeInjectOutputPlan{}, newXcodeInjectUsageError("source is required for copy outputs")
		}
		payload, err := os.ReadFile(sourcePath)
		if err != nil {
			return xcodeInjectOutputPlan{}, err
		}
		return xcodeInjectOutputPlan{Type: outputType, Path: targetPath, Source: sourcePath, Payload: payload}, nil
	default:
		return xcodeInjectOutputPlan{}, newXcodeInjectUsageError("type must be one of plist, json, text, copy")
	}
}

func renderXcodeInjectValue(value any, values map[string]any) (any, error) {
	switch typed := value.(type) {
	case string:
		return renderXcodeInjectString(typed, values)
	case []any:
		rendered := make([]any, 0, len(typed))
		for _, item := range typed {
			renderedItem, err := renderXcodeInjectValue(item, values)
			if err != nil {
				return nil, err
			}
			rendered = append(rendered, renderedItem)
		}
		return rendered, nil
	case map[string]any:
		rendered := make(map[string]any, len(typed))
		for key, item := range typed {
			renderedItem, err := renderXcodeInjectValue(item, values)
			if err != nil {
				return nil, err
			}
			rendered[key] = renderedItem
		}
		return rendered, nil
	default:
		return value, nil
	}
}

func renderXcodeInjectString(input string, values map[string]any) (string, error) {
	rendered := input
	for range len(values) + 1 {
		before := rendered
		for _, key := range sortedXcodeInjectValueKeys(values) {
			placeholder := "${" + key + "}"
			if !strings.Contains(rendered, placeholder) {
				continue
			}
			rendered = strings.ReplaceAll(rendered, placeholder, fmt.Sprint(values[key]))
		}
		if !strings.Contains(rendered, "${") {
			return rendered, nil
		}
		if rendered == before {
			break
		}
	}
	if start := strings.Index(rendered, "${"); start != -1 {
		if end := strings.Index(rendered[start+2:], "}"); end != -1 {
			name := rendered[start+2 : start+2+end]
			return "", newXcodeInjectUsageError("missing value for placeholder %q", name)
		}
		return "", newXcodeInjectUsageError("unclosed placeholder in %q", rendered[start:])
	}
	return rendered, nil
}

func sortedXcodeInjectValueKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j]) || (len(keys[i]) == len(keys[j]) && keys[i] < keys[j])
	})
	return keys
}

func writeXcodeInjectBytes(path, outputType, source string, payload []byte, opts xcodeInjectOptions) (xcodeInjectFileResult, error) {
	result := xcodeInjectFileResult{
		Type:   outputType,
		Path:   path,
		Source: source,
		Bytes:  int64(len(payload)),
	}
	if err := validateXcodeInjectDestination(path, opts.Overwrite); err != nil {
		return xcodeInjectFileResult{}, err
	}
	if opts.DryRun {
		if outputType == "copy" {
			result.Action = "would_copy"
		} else {
			result.Action = "would_write"
		}
		return result, nil
	}
	if _, err := shared.WriteFileNoSymlinkOverwrite(path, bytes.NewReader(payload), 0o644, ".asc-inject-*", ".asc-inject-backup-*"); err != nil {
		return xcodeInjectFileResult{}, err
	}
	if outputType == "copy" {
		result.Action = "copied"
	} else {
		result.Action = "written"
	}
	return result, nil
}

func validateXcodeInjectDestination(path string, overwrite bool) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !overwrite {
		return newXcodeInjectUsageError("output path %q already exists; use --overwrite", path)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to overwrite symlink %q", path)
	}
	if info.IsDir() {
		return fmt.Errorf("output path %q is a directory", path)
	}
	return nil
}

func resolveXcodeInjectPath(baseDir, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(baseDir, path))
}

func xcodeInjectResultRows(result xcodeInjectResult) [][]string {
	rows := make([][]string, 0, len(result.Outputs))
	for _, output := range result.Outputs {
		rows = append(rows, []string{output.Type, output.Action, output.Path, output.Source})
	}
	if len(rows) == 0 {
		return [][]string{{"", "", "", ""}}
	}
	return rows
}
