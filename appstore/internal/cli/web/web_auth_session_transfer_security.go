package web

import (
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

// prepareWebSessionBundleOutput protects the staging inode before the bundle
// bytes are written. On Windows this applies and verifies an owner-only DACL;
// Unix implementations apply the corresponding mode and ACL policy.
func prepareWebSessionBundleOutput(file *os.File) error {
	return secureopen.PreparePrivateFile(file, webSessionBundleFileMode)
}

// createWebSessionBundleStagingFile establishes the platform's private-file
// policy during native creation, before the staging name becomes visible.
func createWebSessionBundleStagingFile(root *os.Root, name string, perm os.FileMode) (*os.File, error) {
	return secureopen.OpenNewPrivateFileNoFollowInRoot(root, name, perm)
}
