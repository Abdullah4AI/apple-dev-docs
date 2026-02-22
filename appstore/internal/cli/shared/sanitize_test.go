package shared

import "testing"

func TestSanitizeTerminal_RemovesControlChars(t *testing.T) {
	input := "ok\x1b[31mred\nmore\x7f"
	got := SanitizeTerminal(input)
	want := "ok[31mredmore"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
