package urlsanitize

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// RedactedPlaceholder replaces a URL that cannot be reduced to a safe form.
const RedactedPlaceholder = "[REDACTED]"

// RedactURLForError reduces a URL to scheme, host, and path. Userinfo, query
// values, and the fragment are dropped because presigned uploads carry their
// capability there. The path is kept because it identifies the operation.
func RedactURLForError(rawURL string) string {
	parsed, ok := parseForError(rawURL)
	if !ok {
		return RedactedPlaceholder
	}
	return parsed.String()
}

// RedactURLHostForError reduces a URL to scheme and host only. Use it when the
// path is itself a credential, as with Slack incoming webhooks.
func RedactURLHostForError(rawURL string) string {
	parsed, ok := parseForError(rawURL)
	if !ok {
		return RedactedPlaceholder
	}
	// parseForError guarantees a hierarchical URL with a host.
	parsed.Path = ""
	parsed.RawPath = ""
	return parsed.String()
}

func parseForError(rawURL string) (*url.URL, bool) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	if parsed.Opaque != "" || parsed.Host == "" {
		// A URL without an authority keeps everything after the scheme in
		// Opaque, which URL.String renders verbatim, and a hostless URL is
		// not the absolute request URL these boundaries expect; nothing in
		// either can be proven safe.
		return nil, false
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""
	return parsed, true
}

// ClassifyTransportFailure returns a short, credential-free description of a
// transport failure so sanitized errors keep their diagnostic value.
func ClassifyTransportFailure(err error) string {
	var dnsError *net.DNSError
	var netError *net.OpError
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded), os.IsTimeout(err):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.As(err, &dnsError):
		return "dns lookup"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection refused"
	case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
		return "network unreachable"
	case errors.As(err, &netError):
		operation := strings.ToLower(netError.Op)
		if strings.Contains(operation, "proxy") {
			return "proxy connection"
		}
		return "network connection"
	default:
		return ""
	}
}

// TransportError describes a failed request without repeating the request URL's
// credentials. Unwrap keeps errors.Is and errors.As usable on the cause.
type TransportError struct {
	Message string
	Err     error
}

// Error returns the sanitized message. The wrapped cause is deliberately not
// interpolated: net/http renders the full request URL in its error text.
func (e *TransportError) Error() string {
	return e.Message
}

// Unwrap exposes the underlying transport error for inspection.
func (e *TransportError) Unwrap() error {
	return e.Err
}

// NewTransportError builds a sanitized transport error for an operation against
// safeURL, which must already have been reduced by one of the redact helpers.
func NewTransportError(operation, safeURL string, err error) *TransportError {
	message := fmt.Sprintf("%s failed for %s", operation, safeURL)
	if class := ClassifyTransportFailure(err); class != "" {
		message += " (" + class + ")"
	}
	return &TransportError{Message: message, Err: err}
}
