package asc

import (
	"strings"
	"testing"
)

func TestSigningResignResultRegisteredTableAndMarkdownOutput(t *testing.T) {
	result := &SigningResignResult{
		SchemaVersion: 1,
		Command:       "signing resign",
		Input: SigningResignInputResult{
			SizeBytes: 42,
			SHA256:    strings.Repeat("A", 64),
		},
		Output: SigningResignArtifactResult{
			Path:      "/safe/output/resigned.ipa",
			SizeBytes: 43,
			SHA256:    strings.Repeat("B", 64),
		},
		Identity: SigningResignIdentityResult{
			CertificateSHA256: strings.Repeat("C", 64),
			TeamID:            "TEAMID",
		},
		Targets: []SigningResignTargetResult{{
			Kind:          "application",
			RelativePath:  "Payload/App.app",
			BundleID:      "com.example.app",
			ProfileClass:  "app-store",
			ProfileUUID:   "PROFILE-UUID",
			ProfileSHA256: strings.Repeat("D", 64),
			Status:        "verified",
		}},
		Verification: SigningResignVerification{Status: "verified", Scope: "complete"},
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[SigningResignResult]()) {
		t.Fatal("SigningResignResult is not registered with the output renderer")
	}

	table := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"schemaVersion", "1", "input.sizeBytes", "42", "output.path", "com.example.app", "verified"} {
		if !strings.Contains(table, want) {
			t.Fatalf("expected table to contain %q, got %q", want, table)
		}
	}
	markdown := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"| schemaVersion", "| input.sizeBytes", "| output.path", "| com.example.app"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got %q", want, markdown)
		}
	}
}
