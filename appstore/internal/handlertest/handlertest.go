// Package handlertest reports test-fixture assertion failures raised by code
// that does not run on the goroutine running the test: httptest handlers,
// RoundTrippers, injected dependency stubs, and worker goroutines.
//
// t.Fatal, t.Fatalf, and t.FailNow may only be called from the goroutine
// running the test. Calling them from an httptest handler skips the response
// write and leaves the client under test waiting for a reply that never
// arrives. Calling them from a worker goroutine terminates that goroutine
// before it can send its result, so the test deadlocks on the channel or wait
// group instead of failing, and the package eventually panics on the test
// timeout with the original failure buried in the dump.
//
// An Asserter records the failure with Errorf, which is safe from any
// goroutine, and returns a value the fixture can hand back to its caller, so
// the code under test observes a deterministic failure and unwinds normally.
package handlertest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

const (
	assertionCode  = "TEST_FIXTURE_ASSERTION"
	assertionTitle = "test fixture assertion failed"
)

// reporter is the subset of testing.TB an Asserter needs. It exists so the
// package can test its own reporting behavior; testing.TB cannot be
// implemented outside the testing package.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// Asserter reports fixture failures from off-test-goroutine code.
type Asserter struct {
	reporter reporter
}

// New returns an Asserter that reports failures to tb.
func New(tb testing.TB) *Asserter {
	tb.Helper()
	return &Asserter{reporter: tb}
}

// Errorf records a failure and returns an error carrying the same message.
//
// Use it wherever the fixture can hand an error back to the code under test:
// a RoundTripper, an injected dependency stub, or a worker goroutine that
// reports through a channel. The message is rendered with fmt.Errorf, so %w
// keeps wrapping the operand and still prints readably in the test log.
//
//	return nil, fixture.Errorf("unexpected request: %s %s", req.Method, req.URL)
func (a *Asserter) Errorf(format string, args ...any) error {
	a.reporter.Helper()

	err := fmt.Errorf(format, args...)
	a.reporter.Errorf("%s", err.Error())
	return err
}

// Respond records a failure and writes an HTTP 500 JSON error body to w.
//
// Use it from httptest handlers, which cannot return an error. The client
// under test receives a complete response instead of blocking until its own
// timeout, and the recorded failure fails the test.
//
//	default:
//	    fixture.Respond(w, "unexpected request: %s %s", req.Method, req.URL.Path)
//	    return
func (a *Asserter) Respond(w http.ResponseWriter, format string, args ...any) {
	a.reporter.Helper()

	body := a.record(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = io.WriteString(w, body)
}

// Response records a failure and returns a synthetic HTTP 500 JSON response.
//
// Use it from fixture helpers that must return an *http.Response and have no
// way to signal an error, such as a response builder called from inside a
// RoundTripper.
//
//	body, err := json.Marshal(payload)
//	if err != nil {
//	    return fixture.Response("marshal territory response: %v", err)
//	}
func (a *Asserter) Response(format string, args ...any) *http.Response {
	a.reporter.Helper()

	body := a.record(format, args...)
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)),
		StatusCode:    http.StatusInternalServerError,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

// record reports the formatted message and returns the JSON error body that
// carries it to the client under test.
func (a *Asserter) record(format string, args ...any) string {
	a.reporter.Helper()

	message := fmt.Errorf(format, args...).Error()
	a.reporter.Errorf("%s", message)
	return errorBody(message)
}

// errorObject mirrors one entry of an App Store Connect error envelope so the
// client under test decodes a fixture failure the same way it decodes a real
// API failure.
type errorObject struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type errorEnvelope struct {
	Errors []errorObject `json:"errors"`
}

func errorBody(message string) string {
	encoded, err := json.Marshal(errorEnvelope{Errors: []errorObject{{
		Status: strconv.Itoa(http.StatusInternalServerError),
		Code:   assertionCode,
		Title:  assertionTitle,
		Detail: message,
	}}})
	if err != nil {
		return fmt.Sprintf(`{"errors":[{"status":"500","code":%q,"title":%q}]}`, assertionCode, assertionTitle)
	}
	return string(encoded)
}
