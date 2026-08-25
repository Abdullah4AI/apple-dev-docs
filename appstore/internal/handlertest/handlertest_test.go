package handlertest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// clientDeadline bounds every request in this package so a regression that
// reintroduces the hang fails the test instead of running until the package
// timeout.
const clientDeadline = 5 * time.Second

// fakeReporter records what an Asserter reports without failing the enclosing
// test. testing.TB cannot be implemented outside the testing package, which is
// why Asserter holds the narrower reporter interface.
type fakeReporter struct {
	mu       sync.Mutex
	messages []string
	helpers  int
}

func (r *fakeReporter) Helper() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.helpers++
}

func (r *fakeReporter) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

func (r *fakeReporter) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.messages...)
}

func newFakeAsserter() (*Asserter, *fakeReporter) {
	recorder := &fakeReporter{}
	return &Asserter{reporter: recorder}, recorder
}

func TestNewReportsThroughTheTestItself(t *testing.T) {
	fixture := New(t)
	if fixture.reporter != reporter(t) {
		t.Fatalf("New() bound %#v, want the test passed to it", fixture.reporter)
	}
}

func TestRespondFailsTheTestAndDeliversFiveHundredWithoutHanging(t *testing.T) {
	fixture, recorder := newFakeAsserter()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fixture.Respond(w, "unexpected request: %s %s", req.Method, req.URL.Path)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), clientDeadline)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/v1/unexpected", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	started := time.Now()
	response, err := server.Client().Do(request)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("client did not receive a response from a failing handler: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if elapsed >= clientDeadline {
		t.Fatalf("failing handler took %s to answer; the client must not wait on an assertion", elapsed)
	}
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	if len(envelope.Errors) != 1 {
		t.Fatalf("error envelope = %+v, want exactly one error", envelope)
	}
	if envelope.Errors[0].Status != "500" || envelope.Errors[0].Code != assertionCode {
		t.Fatalf("error object = %+v", envelope.Errors[0])
	}
	if envelope.Errors[0].Detail != "unexpected request: GET /v1/unexpected" {
		t.Fatalf("error detail = %q", envelope.Errors[0].Detail)
	}

	messages := recorder.snapshot()
	if len(messages) != 1 || messages[0] != "unexpected request: GET /v1/unexpected" {
		t.Fatalf("reported messages = %#v, want the formatted assertion exactly once", messages)
	}
}

func TestErrorfReportsAndLetsAWorkerGoroutineReturn(t *testing.T) {
	fixture, recorder := newFakeAsserter()

	// Models the shape that deadlocks with t.Fatal: a worker that must send its
	// result before the test can proceed.
	results := make(chan error, 1)
	go func() {
		results <- fixture.Errorf("stale creator overwrote completed state")
	}()

	select {
	case err := <-results:
		if err == nil || err.Error() != "stale creator overwrote completed state" {
			t.Fatalf("worker error = %v", err)
		}
	case <-time.After(clientDeadline):
		t.Fatal("worker goroutine never reported; Errorf must not terminate the caller")
	}

	messages := recorder.snapshot()
	if len(messages) != 1 || messages[0] != "stale creator overwrote completed state" {
		t.Fatalf("reported messages = %#v", messages)
	}
}

func TestErrorfKeepsWrappedOperandsReadableAndUnwrappable(t *testing.T) {
	fixture, recorder := newFakeAsserter()
	cause := errors.New("unexpected EOF")

	err := fixture.Errorf("decode PATCH: %w", cause)
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false, want true", err)
	}
	if err.Error() != "decode PATCH: unexpected EOF" {
		t.Fatalf("error = %q", err.Error())
	}

	messages := recorder.snapshot()
	if len(messages) != 1 || messages[0] != "decode PATCH: unexpected EOF" {
		t.Fatalf("reported messages = %#v, want the wrapped operand rendered readably", messages)
	}
}

func TestResponseReportsAndReachesTheClientThroughARoundTripper(t *testing.T) {
	fixture, recorder := newFakeAsserter()

	client := &http.Client{
		Timeout: clientDeadline,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return fixture.Response("marshal territory response: %v", io.ErrUnexpectedEOF), nil
		}),
	}

	response, err := client.Get("https://api.appstoreconnect.apple.com/v1/apps/app-1")
	if err != nil {
		t.Fatalf("client did not receive a response from a failing fixture: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "marshal territory response: unexpected EOF") {
		t.Fatalf("body = %s", body)
	}

	messages := recorder.snapshot()
	if len(messages) != 1 || messages[0] != "marshal territory response: unexpected EOF" {
		t.Fatalf("reported messages = %#v", messages)
	}
}

func TestReportersAreCalledOnceForEveryAssertion(t *testing.T) {
	fixture, recorder := newFakeAsserter()

	_ = fixture.Errorf("first")
	fixture.Respond(httptest.NewRecorder(), "second")
	_ = fixture.Response("third")

	messages := recorder.snapshot()
	want := []string{"first", "second", "third"}
	if len(messages) != len(want) {
		t.Fatalf("reported messages = %#v, want %#v", messages, want)
	}
	for index, message := range want {
		if messages[index] != message {
			t.Fatalf("message %d = %q, want %q", index, messages[index], message)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
