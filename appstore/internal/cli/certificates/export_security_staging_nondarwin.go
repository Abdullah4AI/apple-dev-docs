//go:build !windows && !darwin

package certificates

import (
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

func createCertificateExportStagingFilePlatform(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return secureopen.OpenNewFileNoFollowInRoot(root, name, perm)
}
