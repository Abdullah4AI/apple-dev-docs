package asc

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPrintHumanOutput_XcodeTestIncludesCasesAndFailures(t *testing.T) {
	result := &XcodeTestResult{
		Action: "test",
		Tests: &XcodeTestSummary{
			Total:  2,
			Passed: 1,
			Failed: 1,
			Cases: []XcodeTestCase{
				{Identifier: "DemoTests/Smoke/testPass", Status: "passed"},
				{Identifier: "DemoTests/Smoke/testFail", Status: "failed"},
			},
			Failures: []XcodeTestFailure{{
				Identifier: "DemoTests/Smoke/testFail",
				Message:    "assertion failed",
			}},
		},
	}

	for _, renderer := range []struct {
		name string
		fn   func(any) error
	}{
		{name: "table", fn: PrintTable},
		{name: "markdown", fn: PrintMarkdown},
	} {
		renderer := renderer
		t.Run(renderer.name, func(t *testing.T) {
			assertRenderedNonJSONContains(t, renderer.fn, result,
				"DemoTests/Smoke/testFail", "failed", "assertion failed")
		})
	}
}

func TestFormatXcodeTestFailureTruncatesAtUTF8Boundary(t *testing.T) {
	message := strings.Repeat("a", maxXcodeTestHumanMessage-1) + "é"
	got := formatXcodeTestFailure(XcodeTestFailure{Identifier: "Demo/test", Message: message})
	if !utf8.ValidString(got) {
		t.Fatalf("formatted failure is invalid UTF-8: %q", got)
	}
	if strings.Contains(got, "é") {
		t.Fatalf("formatted failure = %q, want incomplete trailing rune removed", got)
	}
	if !strings.HasSuffix(got, strings.Repeat("a", maxXcodeTestHumanMessage-1)) {
		t.Fatalf("formatted failure does not preserve complete prefix: %q", got)
	}
}

func TestXcodeTestCaseUsesCamelCaseClassNameKey(t *testing.T) {
	data, err := json.Marshal(XcodeTestCase{
		Identifier: "DemoTests/Smoke/testPass",
		Name:       "testPass",
		Classname:  "Smoke",
		Status:     "passed",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), `"className":"Smoke"`) {
		t.Fatalf("XcodeTestCase JSON = %s, want camelCase className key", data)
	}
	if strings.Contains(string(data), `"classname"`) {
		t.Fatalf("XcodeTestCase JSON = %s, want no lowercase classname key", data)
	}
}
