package asc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestLegacy441ClientMethodsHaveDeprecationDocs(t *testing.T) {
	want := map[string]string{
		// The remaining v1 methods are kept only for `asc iap setup`,
		// `asc subscriptions setup`, and `asc validate`.
		"GetInAppPurchaseLocalizations":       "Use GetInAppPurchaseVersionLocalizations with an in-app purchase version ID.",
		"CreateInAppPurchaseLocalization":     "Use CreateInAppPurchaseLocalizationV2 with an in-app purchase version ID.",
		"GetSubscriptionLocalizations":        "Use GetSubscriptionVersionLocalizations with a subscription version ID.",
		"CreateSubscriptionLocalization":      "Use CreateSubscriptionLocalizationV2 with a subscription version ID.",
		"GetSubscriptionImages":               "Use GetSubscriptionVersionImages with a subscription version ID.",
		"GetSubscriptionGroupLocalizations":   "Use GetSubscriptionGroupVersionLocalizations with a subscription group version ID.",
		"CreateSubscriptionGroupLocalization": "Use CreateSubscriptionGroupLocalizationV2 with a subscription group version ID.",
	}
	verified := make(map[string]bool, len(want))
	for method, guidance := range want {
		if strings.TrimSpace(guidance) == "" {
			t.Fatalf("expected replacement guidance for %s must not be empty", method)
		}
	}

	files := []string{
		"client_iap.go",
		"client_iap_subresources.go",
		"client_subscription_resources.go",
		"client_subscriptions.go",
	}
	fset := token.NewFileSet()
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !isExportedClientMethod(function) {
				continue
			}

			guidance, deprecated := deprecatedGuidance(function)
			expected, tracked := want[function.Name.Name]
			if deprecated && !tracked {
				t.Errorf("unexpected deprecated Client method %s; add it to the App Store Connect API 4.4.1 inventory", function.Name.Name)
				continue
			}
			if !tracked {
				continue
			}
			if !deprecated {
				t.Errorf("%s must name its App Store Connect API 4.4.1 replacement in a Deprecated doc comment", function.Name.Name)
				continue
			}
			if guidance != expected {
				t.Errorf("%s deprecation guidance = %q, want %q", function.Name.Name, guidance, expected)
				continue
			}
			verified[function.Name.Name] = true
		}
	}

	for method := range want {
		if !verified[method] {
			t.Errorf("did not verify deprecation docs for %s", method)
		}
	}
}

func isExportedClientMethod(function *ast.FuncDecl) bool {
	if function == nil || function.Recv == nil || !function.Name.IsExported() || len(function.Recv.List) != 1 {
		return false
	}

	receiver := function.Recv.List[0].Type
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		receiver = pointer.X
	}
	identifier, ok := receiver.(*ast.Ident)
	return ok && identifier.Name == "Client"
}

func deprecatedGuidance(function *ast.FuncDecl) (string, bool) {
	if function == nil || function.Doc == nil {
		return "", false
	}

	const marker = "Deprecated:"
	doc := function.Doc.Text()
	index := strings.Index(doc, marker)
	if index < 0 {
		return "", false
	}
	return strings.TrimSpace(doc[index+len(marker):]), true
}
