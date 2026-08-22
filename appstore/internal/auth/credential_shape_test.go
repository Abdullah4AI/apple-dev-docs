package auth

import (
	"strings"
	"testing"
)

func TestLooksLikeIssuerID(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "lowercase canonical uuid", value: "69a6de00-aaaa-bbbb-cccc-123456789abc", want: true},
		{name: "uppercase canonical uuid", value: "A7EFEF21-3432-404F-A488-083800B570FF", want: true},
		{name: "surrounding whitespace", value: "  09f4080c-6ee7-4e52-8103-e1241eaaa58a  ", want: true},
		{name: "empty", value: "", want: false},
		{name: "key id", value: "39MX87M9Y4", want: false},
		{name: "unhyphenated hex", value: "69a6de0000aabbbbccccdd123456789a", want: false},
		{name: "braced uuid", value: "{69a6de00-aaaa-bbbb-cccc-123456789abc}", want: false},
		{name: "urn uuid", value: "urn:uuid:69a6de00-aaaa-bbbb-cccc-123456789abc", want: false},
		{name: "placeholder", value: "issuer-uuid", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := LooksLikeIssuerID(test.value); got != test.want {
				t.Fatalf("LooksLikeIssuerID(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestLooksLikeKeyID(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "typical key id", value: "39MX87M9Y4", want: true},
		{name: "all digits", value: "1234567890", want: true},
		{name: "letters and digits", value: "TEAM123456", want: true},
		{name: "twelve characters", value: "ABC123SECRET", want: true},
		{name: "surrounding whitespace", value: "  39MX87M9Y4  ", want: true},
		{name: "empty", value: "", want: false},
		{name: "issuer uuid", value: "69a6de00-aaaa-bbbb-cccc-123456789abc", want: false},
		{name: "lowercase", value: "39mx87m9y4", want: false},
		{name: "underscore", value: "TEST_KEY_ID", want: false},
		{name: "too short", value: "ENVKEY", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := LooksLikeKeyID(test.value); got != test.want {
				t.Fatalf("LooksLikeKeyID(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestInspectCredentialShapesAcceptsValidAndUnusualCredentials(t *testing.T) {
	labels := CredentialShapeLabels{KeyID: "ASC_KEY_ID", IssuerID: "ASC_ISSUER_ID"}

	for _, test := range []struct {
		name     string
		keyID    string
		issuerID string
	}{
		{name: "typical pair", keyID: "39MX87M9Y4", issuerID: "69a6de00-aaaa-bbbb-cccc-123456789abc"},
		{name: "all digit key id", keyID: "1234567890", issuerID: "69a6de00-aaaa-bbbb-cccc-123456789abc"},
		{name: "lowercase key id", keyID: "39mx87m9y4", issuerID: "69a6de00-aaaa-bbbb-cccc-123456789abc"},
		{name: "mixed case key id", keyID: "39Mx87M9y4", issuerID: "09f4080c-6ee7-4e52-8103-e1241eaaa58a"},
		{name: "short key id", keyID: "KEY123", issuerID: "A7EFEF21-3432-404F-A488-083800B570FF"},
		{name: "underscore key id", keyID: "TEST_KEY_ID", issuerID: "A7EFEF21-3432-404F-A488-083800B570FF"},
		{name: "uppercase issuer uuid", keyID: "39MX87M9Y4", issuerID: "A7EFEF21-3432-404F-A488-083800B570FF"},
		{name: "individual key without issuer", keyID: "39MX87M9Y4", issuerID: ""},
		{name: "nothing provided", keyID: "", issuerID: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if findings := InspectCredentialShapes(labels, test.keyID, test.issuerID); len(findings) != 0 {
				t.Fatalf("InspectCredentialShapes(%q, %q) = %#v, want no findings", test.keyID, test.issuerID, findings)
			}
		})
	}
}

func TestInspectCredentialShapesDetectsSwappedValues(t *testing.T) {
	labels := CredentialShapeLabels{KeyID: "ASC_KEY_ID", IssuerID: "ASC_ISSUER_ID"}
	findings := InspectCredentialShapes(labels, "69a6de00-aaaa-bbbb-cccc-123456789abc", "39MX87M9Y4")

	if len(findings) != 2 {
		t.Fatalf("InspectCredentialShapes() = %#v, want 2 findings", findings)
	}
	if findings[0].Field != "ASC_KEY_ID" {
		t.Fatalf("findings[0].Field = %q, want ASC_KEY_ID", findings[0].Field)
	}
	if findings[0].Message != "ASC_KEY_ID looks like an issuer ID — the values may be swapped" {
		t.Fatalf("findings[0].Message = %q", findings[0].Message)
	}
	if !findings[0].DefiniteSwap {
		t.Fatal("expected findings[0].DefiniteSwap to be true when the key ID is a UUID and the issuer ID is not")
	}
	if findings[1].Field != "ASC_ISSUER_ID" {
		t.Fatalf("findings[1].Field = %q, want ASC_ISSUER_ID", findings[1].Field)
	}
	if !strings.Contains(findings[1].Message, "looks like a key ID") ||
		!strings.Contains(findings[1].Message, "the values may be swapped") {
		t.Fatalf("findings[1].Message = %q, want a key ID swap hint", findings[1].Message)
	}
	for _, finding := range findings {
		if finding.Recommendation == "" {
			t.Fatalf("finding %+v has no recommendation", finding)
		}
	}
}

func TestInspectCredentialShapesStaysUncertainWithoutBothSignals(t *testing.T) {
	labels := CredentialShapeLabels{KeyID: "ASC_KEY_ID", IssuerID: "ASC_ISSUER_ID"}

	t.Run("uuid key id without issuer id", func(t *testing.T) {
		findings := InspectCredentialShapes(labels, "69a6de00-aaaa-bbbb-cccc-123456789abc", "")
		if len(findings) != 1 {
			t.Fatalf("InspectCredentialShapes() = %#v, want 1 finding", findings)
		}
		if findings[0].DefiniteSwap {
			t.Fatal("expected no definite swap when the issuer ID is absent")
		}
		if strings.Contains(findings[0].Message, "swapped") {
			t.Fatalf("findings[0].Message = %q, want no swap hint without both signals", findings[0].Message)
		}
	})

	t.Run("both values are uuids", func(t *testing.T) {
		findings := InspectCredentialShapes(labels, "69a6de00-aaaa-bbbb-cccc-123456789abc", "09f4080c-6ee7-4e52-8103-e1241eaaa58a")
		if len(findings) != 1 {
			t.Fatalf("InspectCredentialShapes() = %#v, want 1 finding", findings)
		}
		if findings[0].DefiniteSwap {
			t.Fatal("expected no definite swap when the issuer ID is a valid UUID")
		}
		if strings.Contains(findings[0].Message, "swapped") {
			t.Fatalf("findings[0].Message = %q, want no swap hint without both signals", findings[0].Message)
		}
	})

	t.Run("uuid key id with generic malformed issuer", func(t *testing.T) {
		findings := InspectCredentialShapes(labels, "69a6de00-aaaa-bbbb-cccc-123456789abc", "issuer-uuid")
		if len(findings) != 2 {
			t.Fatalf("InspectCredentialShapes() = %#v, want 2 findings", findings)
		}
		for _, finding := range findings {
			if finding.DefiniteSwap {
				t.Fatal("expected no definite swap without a key-shaped issuer ID")
			}
			if strings.Contains(finding.Message, "swapped") {
				t.Fatalf("finding.Message = %q, want no swap hint without both signals", finding.Message)
			}
		}
	})

	t.Run("malformed issuer id only", func(t *testing.T) {
		findings := InspectCredentialShapes(labels, "39MX87M9Y4", "ENVISS")
		if len(findings) != 1 {
			t.Fatalf("InspectCredentialShapes() = %#v, want 1 finding", findings)
		}
		if findings[0].Field != "ASC_ISSUER_ID" {
			t.Fatalf("findings[0].Field = %q, want ASC_ISSUER_ID", findings[0].Field)
		}
		if !strings.Contains(findings[0].Message, "is not a UUID") {
			t.Fatalf("findings[0].Message = %q, want a UUID shape message", findings[0].Message)
		}
		if strings.Contains(findings[0].Message, "swapped") {
			t.Fatalf("findings[0].Message = %q, want no swap claim without evidence", findings[0].Message)
		}
		if findings[0].DefiniteSwap {
			t.Fatal("expected no definite swap when only the issuer ID is malformed")
		}
	})
}

func TestInspectCredentialShapesUsesFlagLabels(t *testing.T) {
	labels := CredentialShapeLabels{KeyID: "--key-id", IssuerID: "--issuer-id"}
	findings := InspectCredentialShapes(labels, "69a6de00-aaaa-bbbb-cccc-123456789abc", "39MX87M9Y4")

	if len(findings) != 2 {
		t.Fatalf("InspectCredentialShapes() = %#v, want 2 findings", findings)
	}
	if !strings.HasPrefix(findings[0].Message, "--key-id ") {
		t.Fatalf("findings[0].Message = %q, want a --key-id prefix", findings[0].Message)
	}
	if !strings.HasPrefix(findings[1].Message, "--issuer-id ") {
		t.Fatalf("findings[1].Message = %q, want an --issuer-id prefix", findings[1].Message)
	}
}

func TestInspectCredentialShapesNeverEchoesCredentialValues(t *testing.T) {
	labels := CredentialShapeLabels{KeyID: "ASC_KEY_ID", IssuerID: "ASC_ISSUER_ID"}
	keyID := "09f4080c-6ee7-4e52-8103-e1241eaaa58a"
	issuerID := "ABC123SECRET"

	for _, finding := range InspectCredentialShapes(labels, keyID, issuerID) {
		combined := finding.Message + " " + finding.Recommendation
		if strings.Contains(combined, keyID) {
			t.Fatalf("finding leaked the key ID: %q", combined)
		}
		if strings.Contains(combined, issuerID) {
			t.Fatalf("finding leaked the issuer ID: %q", combined)
		}
	}
}
