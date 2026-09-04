//go:build darwin

package certificates

import (
	"path/filepath"
	"strings"
)

// certificateExportRootPath canonicalizes only the well-known macOS aliases
// whose targets are rooted under /private. Other symlinks remain subject to
// the no-follow parent walk and are rejected as untrusted path components.
func certificateExportRootPath(output string) string {
	for _, alias := range []string{"/tmp", "/var"} {
		if output == alias || strings.HasPrefix(output, alias+string(filepath.Separator)) {
			return filepath.Join("/private", strings.TrimPrefix(output, string(filepath.Separator)))
		}
	}
	return output
}
