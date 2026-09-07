package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebAppDistributionUserMutationResultUsesCamelCaseAndNullableChanged(t *testing.T) {
	result := &WebAppDistributionUserMutationResult{
		AppID:       "app-1",
		RecipientID: "recipient-1",
		AppleID:     "account@example.com",
		Changed:     nil,
		Verified:    false,
		Status:      "uncertain",
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(fields["appId"]) != `"app-1"` || string(fields["recipientId"]) != `"recipient-1"` || string(fields["appleId"]) != `"account@example.com"` || string(fields["changed"]) != "null" || string(fields["verified"]) != "false" || string(fields["status"]) != `"uncertain"` {
		t.Fatalf("unexpected camelCase receipt: %s", encoded)
	}
	for _, field := range []string{"app_id", "recipient_id", "apple_id"} {
		if _, ok := fields[field]; ok {
			t.Fatalf("receipt contains snake_case field %q: %s", field, encoded)
		}
	}
}

func TestWebAppDistributionUserMutationResultUsesRegisteredRenderers(t *testing.T) {
	result := &WebAppDistributionUserMutationResult{
		AppID:       "app-1",
		RecipientID: "recipient-1",
		AppleID:     "account@example.com",
		Changed:     func() *bool { value := true; return &value }(),
		Verified:    true,
		Status:      "created",
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[WebAppDistributionUserMutationResult]()) {
		t.Fatal("WebAppDistributionUserMutationResult is not registered with the output renderer")
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
			output := captureStdout(t, func() error { return renderer.fn(result) })
			for _, want := range []string{"App ID", "Recipient ID", "Apple Account", "Changed", "Verified", "Status", "app-1", "recipient-1", "account@example.com", "true", "created"} {
				if !strings.Contains(output, want) {
					t.Fatalf("output missing %q: %q", want, output)
				}
			}
		})
	}
}
