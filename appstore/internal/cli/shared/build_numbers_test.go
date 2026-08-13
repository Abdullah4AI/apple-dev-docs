package shared

import (
	"strings"
	"testing"
)

func TestParseBuildNumberRejectsNonNumeric(t *testing.T) {
	_, err := parseBuildNumber("1a", "processed build")
	if err == nil {
		t.Fatal("expected error for non-numeric build number")
	}
	if !strings.Contains(err.Error(), "processed build") {
		t.Fatalf("expected error to mention source, got %v", err)
	}
}

func TestParseBuildNumberRejectsEmpty(t *testing.T) {
	_, err := parseBuildNumber(" ", "build upload")
	if err == nil {
		t.Fatal("expected error for empty build number")
	}
}

func TestParseBuildNumberAllowsNumeric(t *testing.T) {
	got, err := parseBuildNumber("42", "processed build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != "42" {
		t.Fatalf("expected 42, got %q", got.String())
	}
}

func TestParseBuildNumberAllowsDotSeparatedNumeric(t *testing.T) {
	got, err := parseBuildNumber("1.2.3", "build upload")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %q", got.String())
	}
}

func TestBuildNumberNextIncrementsLastSegment(t *testing.T) {
	parsed, err := parseBuildNumber("1.2.3", "processed build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	next, err := parsed.Next()
	if err != nil {
		t.Fatalf("unexpected error incrementing build number: %v", err)
	}
	if next.String() != "1.2.4" {
		t.Fatalf("expected next build number 1.2.4, got %q", next.String())
	}
}

func TestBuildNumberCompareUsesNumericComponents(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "dotted numeric order", left: "1.10", right: "1.9", want: 1},
		{name: "missing trailing zero is equal", left: "1.2", right: "1.2.0", want: 0},
		{name: "first component wins", left: "10", right: "9.99", want: 1},
		{name: "lower component", left: "2.3.3", right: "2.3.4", want: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, err := parseBuildNumber(test.left, "left")
			if err != nil {
				t.Fatalf("parse left: %v", err)
			}
			right, err := parseBuildNumber(test.right, "right")
			if err != nil {
				t.Fatalf("parse right: %v", err)
			}
			if got := left.Compare(right); got != test.want {
				t.Fatalf("Compare(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestParseBuildTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "rfc3339", input: "2026-02-10T08:00:00Z"},
		{name: "rfc3339nano", input: "2026-02-10T08:00:00.123456789Z"},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid", input: "2026/02/10", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseBuildTimestamp(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IsZero() {
				t.Fatal("expected non-zero parsed timestamp")
			}
		})
	}
}
