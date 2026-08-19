package reviews

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestReviewUsageErrorsCarryStructuredDiagnostics(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("list review sources: %v", err)
	}

	for _, path := range files {
		if matched, _ := filepath.Match("*_test.go", path); matched {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, node)
			call, ok := node.(*ast.CallExpr)
			if !ok || !isSharedReviewCall(call, "UsageError", "UsageErrorf") {
				return true
			}

			for index := len(stack) - 2; index >= 0; index-- {
				ancestor, ok := stack[index].(*ast.CallExpr)
				if ok && isSharedReviewCall(ancestor, "WithDiagnostic") {
					return true
				}
			}
			position := fileSet.Position(call.Pos())
			t.Errorf("%s:%d: shared usage error lacks structured diagnostic", path, position.Line)
			return true
		})
	}
}

func isSharedReviewCall(call *ast.CallExpr, names ...string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok || identifier.Name != "shared" {
		return false
	}
	for _, name := range names {
		if selector.Sel.Name == name {
			return true
		}
	}
	return false
}
