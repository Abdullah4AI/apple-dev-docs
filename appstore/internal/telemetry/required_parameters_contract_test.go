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
			if !ok {
				return true
			}
			parameterArgument, ok := requiredUsageParameterArgument(call)
			if !ok {
				return true
			}
			literal, ok := parameterArgument.(*ast.BasicLit)
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

func requiredUsageParameterArgument(call *ast.CallExpr) (ast.Expr, bool) {
	var name string
	switch expression := call.Fun.(type) {
	case *ast.Ident:
		name = expression.Name
	case *ast.SelectorExpr:
		name = expression.Sel.Name
	}

	switch name {
	case "MissingRequiredUsageError":
		if len(call.Args) == 1 {
			return call.Args[0], true
		}
	case "metadataRequiredInputError":
		if len(call.Args) == 2 {
			return call.Args[0], true
		}
	default:
		return nil, false
	}
	return nil, false
}
