package shared

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

type stubVersionLocalizationClient struct {
	getResp *asc.AppStoreVersionLocalizationsResponse
	getErr  error

	updateErrs  []error
	updateCalls []asc.AppStoreVersionLocalizationAttributes
}

func (s *stubVersionLocalizationClient) GetAppStoreVersionLocalizations(_ context.Context, _ string, _ ...asc.AppStoreVersionLocalizationsOption) (*asc.AppStoreVersionLocalizationsResponse, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResp, nil
}

func (s *stubVersionLocalizationClient) CreateAppStoreVersionLocalization(_ context.Context, _ string, attrs asc.AppStoreVersionLocalizationAttributes) (*asc.AppStoreVersionLocalizationResponse, error) {
	_ = attrs
	return &asc.AppStoreVersionLocalizationResponse{
		Data: asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
			ID: "created-loc",
		},
	}, nil
}

func (s *stubVersionLocalizationClient) UpdateAppStoreVersionLocalization(_ context.Context, _ string, attrs asc.AppStoreVersionLocalizationAttributes) (*asc.AppStoreVersionLocalizationResponse, error) {
	s.updateCalls = append(s.updateCalls, attrs)
	callIndex := len(s.updateCalls) - 1
	if callIndex < len(s.updateErrs) && s.updateErrs[callIndex] != nil {
		return nil, s.updateErrs[callIndex]
	}
	return &asc.AppStoreVersionLocalizationResponse{
		Data: asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
			ID: "existing-loc",
		},
	}, nil
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	defer func() {
		os.Stderr = oldStderr
	}()

	os.Stderr = writer
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		_ = reader.Close()
		done <- buf.String()
	}()

	fn()
	_ = writer.Close()

	return <-done
}

func TestUploadVersionLocalizations_RetriesWithoutWhatsNewOnInitialReleaseError(t *testing.T) {
	client := &stubVersionLocalizationClient{
		getResp: &asc.AppStoreVersionLocalizationsResponse{
			Data: []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
				{
					ID: "existing-loc",
					Attributes: asc.AppStoreVersionLocalizationAttributes{
						Locale: "en-US",
					},
				},
			},
		},
		updateErrs: []error{
			errors.New("An attribute value is not acceptable for the current resource state. The attribute 'whatsNew' is not editable."),
		},
	}

	valuesByLocale := map[string]map[string]string{
		"en-US": {
			"description": "A description",
			"whatsNew":    "Bug fixes and improvements",
		},
	}

	var (
		results []asc.LocalizationUploadLocaleResult
		err     error
	)
	stderr := captureStderr(t, func() {
		results, err = UploadVersionLocalizations(context.Background(), client, "version-id", valuesByLocale, false)
	})
	if err != nil {
		t.Fatalf("UploadVersionLocalizations() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(client.updateCalls) != 2 {
		t.Fatalf("expected 2 update attempts, got %d", len(client.updateCalls))
	}
	if got := client.updateCalls[0].WhatsNew; got != "Bug fixes and improvements" {
		t.Fatalf("expected first attempt to include whatsNew, got %q", got)
	}
	if got := client.updateCalls[1].WhatsNew; got != "" {
		t.Fatalf("expected retry attempt to clear whatsNew, got %q", got)
	}
	if !strings.Contains(stderr, "Retrying without it") {
		t.Fatalf("expected retry warning in stderr, got %q", stderr)
	}
}

func TestUploadVersionLocalizations_DoesNotRetryWhenErrorIsUnrelated(t *testing.T) {
	client := &stubVersionLocalizationClient{
		getResp: &asc.AppStoreVersionLocalizationsResponse{
			Data: []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
				{
					ID: "existing-loc",
					Attributes: asc.AppStoreVersionLocalizationAttributes{
						Locale: "en-US",
					},
				},
			},
		},
		updateErrs: []error{errors.New("network timeout")},
	}

	valuesByLocale := map[string]map[string]string{
		"en-US": {
			"description": "A description",
			"whatsNew":    "Bug fixes and improvements",
		},
	}

	_, err := UploadVersionLocalizations(context.Background(), client, "version-id", valuesByLocale, false)
	if err == nil {
		t.Fatal("expected upload error")
	}
	if len(client.updateCalls) != 1 {
		t.Fatalf("expected a single update attempt, got %d", len(client.updateCalls))
	}
}

func TestUploadVersionLocalizations_DoesNotRetryWhenWhatsNewIsEmpty(t *testing.T) {
	client := &stubVersionLocalizationClient{
		getResp: &asc.AppStoreVersionLocalizationsResponse{
			Data: []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
				{
					ID: "existing-loc",
					Attributes: asc.AppStoreVersionLocalizationAttributes{
						Locale: "en-US",
					},
				},
			},
		},
		updateErrs: []error{
			errors.New("The attribute 'whatsNew' is not editable for this version"),
		},
	}

	valuesByLocale := map[string]map[string]string{
		"en-US": {
			"description": "A description",
		},
	}

	_, err := UploadVersionLocalizations(context.Background(), client, "version-id", valuesByLocale, false)
	if err == nil {
		t.Fatal("expected upload error")
	}
	if len(client.updateCalls) != 1 {
		t.Fatalf("expected a single update attempt, got %d", len(client.updateCalls))
	}
}

func TestIsWhatsNewUnsupportedError(t *testing.T) {
	apiErr := &asc.APIError{
		Title:  "ENTITY_ERROR.ATTRIBUTE.INVALID",
		Detail: "The attribute 'whatsNew' cannot be set for this resource state.",
	}
	if !isWhatsNewUnsupportedError(apiErr) {
		t.Fatal("expected API error with whatsNew detail to be recognized")
	}
	if isWhatsNewUnsupportedError(errors.New("timeout")) {
		t.Fatal("did not expect unrelated error to match")
	}
}
