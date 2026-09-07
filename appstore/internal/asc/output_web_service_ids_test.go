package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebServiceIDMutationUsesRegistry(t *testing.T) {
	result := &WebServiceIDMutationResult{
		Operation:  "rename",
		ServiceID:  "service-1",
		Identifier: "com.example.service",
		Name:       "New Name",
		Changed:    true,
		Verified:   true,
		Status:     "renamed",
	}
	for _, render := range []func(any) error{PrintTable, PrintMarkdown} {
		output := captureStdout(t, func() error { return render(result) })
		for _, want := range []string{"Operation", "Service ID", "rename", "service-1", "com.example.service", "New Name", "true", "renamed"} {
			if !strings.Contains(output, want) {
				t.Fatalf("output missing %q: %q", want, output)
			}
		}
	}
}
