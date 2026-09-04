//go:build darwin

package secureopen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func protectedTestACL(t *testing.T, path string) bool {
	t.Helper()
	output, err := exec.Command("/bin/ls", "-le", path).CombinedOutput()
	if err != nil {
		t.Fatalf("ls -le %q: %v (%s)", path, err, strings.TrimSpace(string(output)))
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.HasSuffix(fields[0], ":") {
			return true
		}
	}
	return false
}

func TestOpenNewFileNoFollowInRootNoInheritIsProtectedAtFirstVisibility(t *testing.T) {
	directory := t.TempDir()
	output, err := exec.Command("/bin/chmod", "+a", "everyone allow read,file_inherit", directory).CombinedOutput()
	if err != nil {
		t.Skipf("cannot apply inheritable test ACL: %v (%s)", err, strings.TrimSpace(string(output)))
	}

	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	observer := func(file *os.File) {
		if protectedTestACL(t, file.Name()) {
			t.Fatalf("new file carried inherited ACL before rooted verification returned")
		}
		info, statErr := file.Stat()
		if statErr != nil {
			t.Fatalf("Stat() during first-visibility observation: %v", statErr)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("new file mode at first visibility = %#o, want 0600", got)
		}
	}

	file, err := openNewFileNoFollowInRootNoInherit(root, "first-visible", 0o600, observer)
	if err != nil {
		t.Fatalf("OpenNewFileNoFollowInRootNoInherit() error = %v", err)
	}
	defer file.Close()
	if err := root.Remove(filepath.Base(file.Name())); err != nil {
		t.Fatalf("remove staged file: %v", err)
	}
}

func TestOpenNewFileNoFollowInRootNoInheritRejectsBroadMode(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	if _, err := OpenNewFileNoFollowInRootNoInherit(root, "broad", 0o640); err == nil {
		t.Fatal("OpenNewFileNoFollowInRootNoInherit() accepted a group-readable mode")
	}
}
