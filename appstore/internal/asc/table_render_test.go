package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHumanRenderersSanitizeEveryHeaderAndCell(t *testing.T) {
	renderers := []struct {
		name   string
		render func([]string, [][]string)
	}{
		{name: "table", render: RenderTable},
		{name: "markdown", render: RenderMarkdown},
	}

	for _, renderer := range renderers {
		t.Run(renderer.name, func(t *testing.T) {
			output := captureStdout(t, func() error {
				renderer.render(
					[]string{"unsafe\x1b]0;title\x07header"},
					[][]string{{"one\u2028two", "bad\xc2byte", "bidi\u202etext"}},
				)
				return nil
			})

			for _, unsafe := range []string{"\x1b", "\u2028", "\u202e", "\xc2"} {
				if strings.Contains(output, unsafe) {
					t.Fatalf("%s output still contains unsafe source bytes %q: %q", renderer.name, unsafe, output)
				}
			}
			for _, want := range []string{"unsafe]0;titleheader", "one two", "bad�byte", "biditext"} {
				if !strings.Contains(output, want) {
					t.Fatalf("%s output = %q, want sanitized value %q", renderer.name, output, want)
				}
			}
		})
	}
}

func TestCentralHumanSanitizationDoesNotChangeJSON(t *testing.T) {
	value := struct {
		Text string `json:"text"`
	}{Text: "raw\x1b\u202e\u2028value"}

	output := captureStdout(t, func() error {
		return PrintJSON(value)
	})

	var decoded struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if decoded.Text != value.Text {
		t.Fatalf("JSON text = %q, want exact original %q", decoded.Text, value.Text)
	}
}
