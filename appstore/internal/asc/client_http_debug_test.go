package asc

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestSanitizeAuthHeader(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: ""},
		{name: "bearer", value: "Bearer token", want: "Bearer [REDACTED]"},
		{name: "basic", value: "Basic abc123", want: "[REDACTED]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sanitizeAuthHeader(test.value); got != test.want {
				t.Fatalf("sanitizeAuthHeader(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestSanitizeURLForLog_RedactsSignedQuery(t *testing.T) {
	rawURL := "https://example.com/path?X-Amz-Signature=abc&foo=bar"
	got := sanitizeURLForLog(rawURL)

	if strings.Contains(got, "X-Amz-Signature=abc") {
		t.Fatalf("expected signature to be redacted, got %q", got)
	}
	if strings.Contains(got, "foo=bar") {
		t.Fatalf("expected non-sensitive values to be redacted for signed URLs, got %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected redacted placeholder in %q", got)
	}
}

func TestSanitizeURLForLog_RedactsTokenQuery(t *testing.T) {
	rawURL := "https://example.com/path?token=abc&foo=bar"
	got := sanitizeURLForLog(rawURL)

	if strings.Contains(got, "token=abc") {
		t.Fatalf("expected token to be redacted, got %q", got)
	}
	if !strings.Contains(got, "foo=bar") {
		t.Fatalf("expected non-sensitive values to remain, got %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected redacted placeholder in %q", got)
	}
}

func TestSanitizeURLForLog_EmptySignatureDoesNotRedactAll(t *testing.T) {
	rawURL := "https://example.com/path?X-Amz-Signature=&foo=bar"
	got := sanitizeURLForLog(rawURL)

	if !strings.Contains(got, "foo=bar") {
		t.Fatalf("expected non-sensitive values to remain, got %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected signature to be redacted, got %q", got)
	}
}

func TestSanitizeURLForLog_RedactsUserInfo(t *testing.T) {
	rawURL := "https://user:pass@example.com/path"
	got := sanitizeURLForLog(rawURL)

	if strings.Contains(got, "user:pass") {
		t.Fatalf("expected userinfo to be redacted, got %q", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected redacted placeholder in %q", got)
	}
}

func TestDebugLoggingRedactsSignedQuery(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := debugLogger
	debugLogger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))
	t.Cleanup(func() { debugLogger = originalLogger })

	debugEnabled := true
	SetDebugOverride(&debugEnabled)
	SetDebugHTTPOverride(&debugEnabled)
	t.Cleanup(func() {
		SetDebugOverride(nil)
		SetDebugHTTPOverride(nil)
	})

	client := newTestClient(t, nil, jsonResponse(http.StatusOK, `{"data":[]}`))
	_, err := client.doOnce(context.Background(), http.MethodGet, "https://example.com/path?X-Amz-Signature=abc&foo=bar", nil)
	if err != nil {
		t.Fatalf("doOnce() error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "X-Amz-Signature=abc") {
		t.Fatalf("expected signature to be redacted, got %q", output)
	}
	if strings.Contains(output, "foo=bar") {
		t.Fatalf("expected signed query values to be redacted, got %q", output)
	}
	if !strings.Contains(output, "REDACTED") {
		t.Fatalf("expected redacted placeholder in %q", output)
	}
}
