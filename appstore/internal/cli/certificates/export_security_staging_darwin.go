//go:build darwin

package certificates

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

// createCertificateExportStagingFilePlatform creates the staging inode with an
// explicit empty, non-inheriting ACL as part of the Darwin kernel create
// operation. The first visible inode is the generated staging name with the
// owner-only policy already applied.
func createCertificateExportStagingFilePlatform(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	if root == nil {
		return nil, fmt.Errorf("create protected output: missing rooted parent")
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || filepath.Clean(name) != name {
		return nil, fmt.Errorf("create protected output: invalid generated name")
	}
	return secureopen.OpenNewFileNoFollowInRootNoInherit(root, name, perm)
}
