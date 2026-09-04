package asc

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
)

// forbiddenAscImports lists first-party packages that internal/asc must not
// import. internal/xcode imports internal/asc so its local Xcode results can
// use the registered camelCase output types, so the reverse edge would create
// an import cycle. That cycle is invisible to a build of this branch alone and
// only appears once both directions exist, so assert the invariant directly.
var forbiddenAscImports = map[string]string{
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/xcode": "internal/xcode imports internal/asc for its registered output types; keep output_*.go free of the domain type and convert in the command layer",
}

func TestAscOutputPackageDoesNotImportDomainPackagesThatImportIt(t *testing.T) {
	packageRoot, err := rootfs.New(".")
	if err != nil {
		t.Fatalf("open package root: %v", err)
	}
	defer packageRoot.Close()

	// Every file read below goes through the rooted handle. The enumeration
	// itself stays on os.ReadDir because rootfs exposes no directory-listing
	// primitive; the names it yields are re-resolved through the root before
	// any bytes are read.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	fileSet := token.NewFileSet()
	inspected := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		source, readErr := packageRoot.ReadFile(name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		file, parseErr := parser.ParseFile(fileSet, name, source, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", name, parseErr)
		}
		inspected++
		for _, importSpec := range file.Imports {
			path, unquoteErr := strconv.Unquote(importSpec.Path.Value)
			if unquoteErr != nil {
				t.Fatalf("unquote import in %s: %v", name, unquoteErr)
			}
			if reason, forbidden := forbiddenAscImports[path]; forbidden {
				t.Errorf("%s imports %s: %s", name, path, reason)
			}
		}
	}

	if inspected == 0 {
		t.Fatal("no package sources inspected; the boundary check would pass vacuously")
	}
}
