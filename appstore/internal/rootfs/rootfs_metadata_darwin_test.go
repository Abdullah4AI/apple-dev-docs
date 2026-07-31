//go:build darwin

package rootfs

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteFilePreservingModePreservesAccessControlList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(path, []byte("original"), 0o640); err != nil {
		t.Fatal(err)
	}

	if output, err := exec.Command("chmod", "+a", "everyone allow read", path).CombinedOutput(); err != nil {
		t.Fatalf("add ACL: %v\n%s", err, output)
	}
	before, beforeOutput := readDarwinACLActions(t, path)
	if before["allow read"] == 0 {
		t.Fatalf("ACL setup did not add an allow-read entry:\n%s", beforeOutput)
	}

	root := mustRoot(t, dir)
	if err := root.WriteFilePreservingMode("existing.txt", []byte("replacement"), 0o644); err != nil {
		t.Fatalf("WriteFilePreservingMode() error = %v", err)
	}
	after, afterOutput := readDarwinACLActions(t, path)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("ACL actions changed across replacement:\nbefore:\n%s\nafter:\n%s", beforeOutput, afterOutput)
	}
}

func readDarwinACLActions(t *testing.T, path string) (map[string]int, []byte) {
	t.Helper()
	output, err := exec.Command("ls", "-lde", path).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect ACL: %v\n%s", err, output)
	}

	actions := make(map[string]int)
	for line := range strings.Lines(string(output)) {
		for _, action := range []string{"allow ", "deny "} {
			if index := strings.Index(line, action); index >= 0 {
				actions[strings.TrimSpace(line[index:])]++
				break
			}
		}
	}
	return actions, output
}
