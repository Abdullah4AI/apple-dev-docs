package releasenotes

import "testing"

func TestFormatNotes_Plain(t *testing.T) {
	notes, err := FormatNotes([]Commit{
		{SHA: "a1b2c3d", Subject: "feat: add thing"},
		{SHA: "d4e5f6g", Subject: "fix: bug"},
	}, "plain")
	if err != nil {
		t.Fatalf("FormatNotes error: %v", err)
	}
	want := "- feat: add thing\n- fix: bug"
	if notes != want {
		t.Fatalf("notes = %q, want %q", notes, want)
	}
}

func TestTruncateNotes_KeepsWholeLinesWhenPossible(t *testing.T) {
	in := "- first\n- second\n- third"
	out, truncated := TruncateNotes(in, len("- first\n- second"))
	if !truncated {
		t.Fatalf("expected truncated=true")
	}
	if out != "- first\n- second" {
		t.Fatalf("out = %q, want %q", out, "- first\n- second")
	}
}

func TestTruncateNotes_TruncatesWithinLineWhenNecessary(t *testing.T) {
	in := "- this is a very long line"
	out, truncated := TruncateNotes(in, 10)
	if !truncated {
		t.Fatalf("expected truncated=true")
	}
	if out != "- this is " {
		t.Fatalf("out = %q, want %q", out, "- this is ")
	}
}

func TestTruncateNotes_ZeroMaxChars(t *testing.T) {
	out, truncated := TruncateNotes("- first\n- second", 0)
	if !truncated {
		t.Fatalf("expected truncated=true")
	}
	if out != "" {
		t.Fatalf("out = %q, want empty string", out)
	}
}

func TestTruncateNotes_ZeroMaxChars_EmptyInput(t *testing.T) {
	out, truncated := TruncateNotes("", 0)
	if truncated {
		t.Fatalf("expected truncated=false")
	}
	if out != "" {
		t.Fatalf("out = %q, want empty string", out)
	}
}
