package shared

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestRequiredUsageCallSitesDeclareParameterIntent(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate required diagnostics contract test")
	}
	cliDir := filepath.Dir(filepath.Dir(currentFile))
	fset := token.NewFileSet()
	var missing []string

	err := filepath.WalkDir(cliDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 0 || !isMissingRequiredUsageErrorCall(call.Fun) {
				return true
			}
			position := fset.Position(call.Pos())
			relative, err := filepath.Rel(cliDir, position.Filename)
			if err != nil {
				relative = position.Filename
			}
			missing = append(missing, fmt.Sprintf("%s:%d", relative, position.Line))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan required-input diagnostics: %v", err)
	}
	if len(missing) == 0 {
		return
	}

	sort.Strings(missing)
	const reportLimit = 25
	reported := missing
	if len(reported) > reportLimit {
		reported = reported[:reportLimit]
	}
	t.Fatalf(
		"found %d MissingRequiredUsageError calls without explicit parameter intent; pass a public flag name or an explicit empty string for multi-parameter requirements:\n%s",
		len(missing),
		strings.Join(reported, "\n"),
	)
}

func isMissingRequiredUsageErrorCall(function ast.Expr) bool {
	switch expression := function.(type) {
	case *ast.Ident:
		return expression.Name == "MissingRequiredUsageError"
	case *ast.SelectorExpr:
		return expression.Sel.Name == "MissingRequiredUsageError"
	default:
		return false
	}
}
