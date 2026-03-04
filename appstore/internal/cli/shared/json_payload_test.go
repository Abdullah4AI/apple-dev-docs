package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJSONFilePayload(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "payload.json")
		if err := os.WriteFile(path, []byte(`{"name":"demo"}`), 0o600); err != nil {
			t.Fatalf("write payload: %v", err)
		}

		payload, err := ReadJSONFilePayload(path)
		if err != nil {
			t.Fatalf("ReadJSONFilePayload unexpected error: %v", err)
		}
		if string(payload) != `{"name":"demo"}` {
			t.Fatalf("unexpected payload: %q", string(payload))
		}
	})

	t.Run("symlink payload", func(t *testing.T) {
		dir := t.TempDir()
		targetPath := filepath.Join(dir, "target.json")
		linkPath := filepath.Join(dir, "payload-link.json")
		if err := os.WriteFile(targetPath, []byte(`{"name":"linked"}`), 0o600); err != nil {
			t.Fatalf("write payload: %v", err)
		}
		if err := os.Symlink(targetPath, linkPath); err != nil {
			t.Skipf("symlink not available in this environment: %v", err)
		}

		payload, err := ReadJSONFilePayload(linkPath)
		if err != nil {
			t.Fatalf("ReadJSONFilePayload unexpected error for symlink: %v", err)
		}
		if string(payload) != `{"name":"linked"}` {
			t.Fatalf("unexpected payload: %q", string(payload))
		}
	})

	t.Run("directory path", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ReadJSONFilePayload(dir)
		if err == nil {
			t.Fatal("expected error for directory payload path")
		}
		if !strings.Contains(err.Error(), "payload path must be a file") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.json")
		if err := os.WriteFile(path, []byte(" \n\t"), 0o600); err != nil {
			t.Fatalf("write payload: %v", err)
		}

		_, err := ReadJSONFilePayload(path)
		if err == nil {
			t.Fatal("expected error for empty payload")
		}
		if !strings.Contains(err.Error(), "payload file is empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.json")
		if err := os.WriteFile(path, []byte(`{"name"`), 0o600); err != nil {
			t.Fatalf("write payload: %v", err)
		}

		_, err := ReadJSONFilePayload(path)
		if err == nil {
			t.Fatal("expected error for invalid payload")
		}
		if !strings.Contains(err.Error(), "invalid JSON:") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
