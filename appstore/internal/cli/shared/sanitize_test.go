package shared

import "testing"

func TestSanitizeTerminal_RemovesControlChars(t *testing.T) {
	input := "ok\x1b[31mred\nmore\x7f"
	got := SanitizeTerminal(input)
	want := "ok[31mred more"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
