package distribution

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIPAAppSourceMaterializesSelectedMainAppOnlyOnRequest(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	path := writeOrderedIPA(t, []orderedZipEntry{
		{Name: "Payload/", Mode: os.ModeDir | 0o755},
		{Name: "Payload/Decoy.app/", Mode: os.ModeDir | 0o755},
		{Name: "Payload/Decoy.app/Info.plist", Mode: os.ModeDir | 0o755},
		{Name: "Payload/Demo.app/", Mode: os.ModeDir | 0o755},
		{Name: "Payload/Demo.app/Info.plist", Data: infoPlist(t, "com.example.demo")},
		{Name: "Payload/Demo.app/Demo", Data: []byte("executable"), Mode: 0o755},
		{Name: "Payload/Demo.app/embedded.mobileprovision", Data: signedProfile(t, profileFixture{
			BundleID: "com.example.demo", Devices: []string{"DEVICE_CANARY"}, Expires: time.Now().Add(time.Hour),
		})},
		{Name: "Symbols/ignored.bin", Data: []byte("outside app")},
	})
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sourceBefore, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	source, err := OpenIPAAppSourceContext(t.Context(), file, info.Size(), InspectOptions{IncludeDevices: true})
	if err != nil {
		t.Fatalf("OpenIPAAppSourceContext() error = %v", err)
	}
	defer source.Cleanup()
	if extracted, globErr := filepath.Glob(filepath.Join(os.TempDir(), ".asc-xcode-install-app-*")); globErr != nil || len(extracted) != 0 {
		t.Fatalf("app payload extracted before MaterializeApp: %v %v", extracted, globErr)
	}
	materialized, err := source.MaterializeApp(t.Context())
	if err != nil {
		t.Fatalf("MaterializeApp() error = %v", err)
	}
	if materialized.Path == "" || filepath.Base(materialized.Path) != "Demo.app" {
		t.Fatalf("materialized path = %q", materialized.Path)
	}
	executable, err := os.Stat(filepath.Join(materialized.Path, "Demo"))
	if err != nil {
		t.Fatal(err)
	}
	if executable.Mode().Perm() != 0o755 {
		t.Fatalf("executable mode = %o, want 755", executable.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(materialized.Path, "..", "ignored.bin")); !os.IsNotExist(err) {
		t.Fatalf("outside app unexpectedly materialized: %v", err)
	}
	pathToRemove := materialized.Path
	materialized.Cleanup()
	if _, err := os.Stat(pathToRemove); !os.IsNotExist(err) {
		t.Fatalf("materialized app still exists after cleanup: %v", err)
	}
	sourceAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceBefore, sourceAfter) {
		t.Fatal("source IPA changed during materialization")
	}
	source.Cleanup()
	if _, err := source.MaterializeApp(t.Context()); err == nil {
		t.Fatal("MaterializeApp() after Cleanup succeeded, want closed-source error")
	}
}
