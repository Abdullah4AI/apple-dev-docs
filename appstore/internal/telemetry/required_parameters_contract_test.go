package telemetry

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestRequiredUsageLiteralParametersAreAllowlisted(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate required parameter telemetry contract test")
	}
	cliDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "cli"))
	fset := token.NewFileSet()
	var rejected []string

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
			if !ok || len(call.Args) != 1 || !isRequiredUsageCall(call.Fun) {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			parameter, err := strconv.Unquote(literal.Value)
			if err != nil || parameter == "" {
				return true
			}
			if sanitizeFailureParameter(parameter) == parameter {
				return true
			}
			position := fset.Position(call.Pos())
			relative, relErr := filepath.Rel(cliDir, position.Filename)
			if relErr != nil {
				relative = position.Filename
			}
			rejected = append(rejected, fmt.Sprintf("%s:%d %s", relative, position.Line, parameter))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan required-input telemetry parameters: %v", err)
	}
	if len(rejected) > 0 {
		t.Fatalf("required-input parameters rejected by telemetry allowlist:\n%s", strings.Join(rejected, "\n"))
	}
}

func isRequiredUsageCall(function ast.Expr) bool {
	switch expression := function.(type) {
	case *ast.Ident:
		return expression.Name == "MissingRequiredUsageError"
	case *ast.SelectorExpr:
		return expression.Sel.Name == "MissingRequiredUsageError"
	default:
		return false
	}
}
