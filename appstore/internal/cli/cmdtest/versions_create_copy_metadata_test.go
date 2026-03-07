package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionsCreateCopyMetadataAppliesSelectedFields(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.Method != http.MethodPost || req.URL.Path != "/v1/appStoreVersions" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			body := `{"data":{"type":"appStoreVersions","id":"ver-new","attributes":{"platform":"IOS","versionString":"2.4.0","appVersionState":"PREPARE_FOR_SUBMISSION"}}}`
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/apps/app-1/appStoreVersions" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			if req.URL.Query().Get("filter[versionString]") != "2.3.2" {
				t.Fatalf("expected filter[versionString]=2.3.2, got %q", req.URL.Query().Get("filter[versionString]"))
			}
			if req.URL.Query().Get("filter[platform]") != "IOS" {
				t.Fatalf("expected filter[platform]=IOS, got %q", req.URL.Query().Get("filter[platform]"))
			}
			body := `{"data":[{"type":"appStoreVersions","id":"ver-source","attributes":{"platform":"IOS","versionString":"2.3.2","appVersionState":"READY_FOR_SALE"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 3:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/ver-source/appStoreVersionLocalizations" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			body := `{"data":[{"type":"appStoreVersionLocalizations","id":"src-en","attributes":{"locale":"en-US","description":"English description","keywords":"en,keywords","whatsNew":"ignored"}},{"type":"appStoreVersionLocalizations","id":"src-fr","attributes":{"locale":"fr-FR","description":"Description FR","keywords":"fr,motscles","whatsNew":"ignored"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 4:
			if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/ver-new/appStoreVersionLocalizations" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			body := `{"data":[{"type":"appStoreVersionLocalizations","id":"dst-en","attributes":{"locale":"en-US"}},{"type":"appStoreVersionLocalizations","id":"dst-fr","attributes":{"locale":"fr-FR"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 5:
			if req.Method != http.MethodPatch || req.URL.Path != "/v1/appStoreVersionLocalizations/dst-en" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			payload, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read payload: %v", err)
			}
			bodyText := string(payload)
			if !strings.Contains(bodyText, `"description":"English description"`) {
				t.Fatalf("expected description in payload, got %s", bodyText)
			}
			if !strings.Contains(bodyText, `"keywords":"en,keywords"`) {
				t.Fatalf("expected keywords in payload, got %s", bodyText)
			}
			if strings.Contains(bodyText, `"whatsNew"`) {
				t.Fatalf("did not expect whatsNew in payload, got %s", bodyText)
			}
			body := `{"data":{"type":"appStoreVersionLocalizations","id":"dst-en","attributes":{"locale":"en-US","description":"English description","keywords":"en,keywords"}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 6:
			if req.Method != http.MethodPatch || req.URL.Path != "/v1/appStoreVersionLocalizations/dst-fr" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			payload, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read payload: %v", err)
			}
			bodyText := string(payload)
			if !strings.Contains(bodyText, `"description":"Description FR"`) {
				t.Fatalf("expected description in payload, got %s", bodyText)
			}
			if !strings.Contains(bodyText, `"keywords":"fr,motscles"`) {
				t.Fatalf("expected keywords in payload, got %s", bodyText)
			}
			if strings.Contains(bodyText, `"whatsNew"`) {
				t.Fatalf("did not expect whatsNew in payload, got %s", bodyText)
			}
			body := `{"data":{"type":"appStoreVersionLocalizations","id":"dst-fr","attributes":{"locale":"fr-FR","description":"Description FR","keywords":"fr,motscles"}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request count %d", requestCount)
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"versions", "create",
			"--app", "app-1",
			"--version", "2.4.0",
			"--platform", "IOS",
			"--copy-metadata-from", "2.3.2",
			"--copy-fields", "description,keywords,whatsNew",
			"--exclude-fields", "whatsNew",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"ver-new"`) {
		t.Fatalf("expected created version id in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"metadataCopy"`) {
		t.Fatalf("expected metadata copy summary in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"copiedLocales":2`) {
		t.Fatalf("expected copied locale count in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"copiedFieldUpdates":4`) {
		t.Fatalf("expected copied field count in output, got %q", stdout)
	}
}

func TestVersionsCreateCopyMetadataSkipsSourceLocalesMissingOnDestination(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			body := `{"data":{"type":"appStoreVersions","id":"ver-new","attributes":{"platform":"IOS","versionString":"2.4.0","appVersionState":"PREPARE_FOR_SUBMISSION"}}}`
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			body := `{"data":[{"type":"appStoreVersions","id":"ver-source","attributes":{"platform":"IOS","versionString":"2.3.2","appVersionState":"READY_FOR_SALE"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 3:
			body := `{"data":[{"type":"appStoreVersionLocalizations","id":"src-en","attributes":{"locale":"en-US","description":"English description"}},{"type":"appStoreVersionLocalizations","id":"src-fr","attributes":{"locale":"fr-FR","description":"Description FR"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 4:
			body := `{"data":[{"type":"appStoreVersionLocalizations","id":"dst-en","attributes":{"locale":"en-US"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 5:
			if req.Method != http.MethodPatch || req.URL.Path != "/v1/appStoreVersionLocalizations/dst-en" {
				t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			}
			body := `{"data":{"type":"appStoreVersionLocalizations","id":"dst-en","attributes":{"locale":"en-US","description":"English description"}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request count %d", requestCount)
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"versions", "create",
			"--app", "app-1",
			"--version", "2.4.0",
			"--platform", "IOS",
			"--copy-metadata-from", "2.3.2",
			"--copy-fields", "description",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stderr, "Warning: skipped source locales not enabled on destination: fr-FR") {
		t.Fatalf("expected skip warning in stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"copiedLocales":1`) {
		t.Fatalf("expected copied locale count in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"skippedLocales":["fr-FR"]`) {
		t.Fatalf("expected skipped locale summary in output, got %q", stdout)
	}
}

func TestVersionsCreateCopyMetadataRejectsInvalidCopyField(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"versions", "create",
			"--app", "app-1",
			"--version", "2.4.0",
			"--copy-metadata-from", "2.3.2",
			"--copy-fields", "description,invalidField",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--copy-fields must be one of:") {
		t.Fatalf("expected copy-fields usage error, got %q", stderr)
	}
}

func TestVersionsCreateCopyMetadataRequiresCopySource(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"versions", "create",
			"--app", "app-1",
			"--version", "2.4.0",
			"--copy-fields", "description",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Error: --copy-metadata-from is required when using --copy-fields or --exclude-fields") {
		t.Fatalf("expected copy source usage error, got %q", stderr)
	}
}

func TestVersionsCreateCopyMetadataRejectsInvalidExcludeField(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"versions", "create",
			"--app", "app-1",
			"--version", "2.4.0",
			"--copy-metadata-from", "2.3.2",
			"--exclude-fields", "description,invalidField",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp usage error, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--exclude-fields must be one of:") {
		t.Fatalf("expected exclude-fields usage error, got %q", stderr)
	}
}
