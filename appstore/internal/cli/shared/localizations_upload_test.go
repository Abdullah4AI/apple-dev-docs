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
	getResp  *asc.AppStoreVersionLocalizationsResponse
	getErr   error
	getCalls int

	createCalls []asc.AppStoreVersionLocalizationAttributes
	updateErrs  []error
	updateCalls []asc.AppStoreVersionLocalizationAttributes
}

func (s *stubVersionLocalizationClient) GetAppStoreVersionLocalizations(_ context.Context, _ string, _ ...asc.AppStoreVersionLocalizationsOption) (*asc.AppStoreVersionLocalizationsResponse, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResp, nil
}

func (s *stubVersionLocalizationClient) CreateAppStoreVersionLocalization(_ context.Context, _ string, attrs asc.AppStoreVersionLocalizationAttributes) (*asc.AppStoreVersionLocalizationResponse, error) {
	s.createCalls = append(s.createCalls, attrs)
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

func TestUploadVersionLocalizationsWithWarnings_RejectsOverLimitKeywordCharactersBeforeFetch(t *testing.T) {
	client := &stubVersionLocalizationClient{}

	valuesByLocale := map[string]map[string]string{
		"ja": {
			"description": "日本語説明",
			"keywords":    strings.Repeat("語", 101),
		},
	}

	_, _, err := UploadVersionLocalizationsWithWarnings(context.Background(), client, "version-id", valuesByLocale, true, SubmitReadinessOptions{})
	if err == nil {
		t.Fatal("expected upload validation error")
	}
	if !strings.Contains(err.Error(), "keywords exceed 100 characters") {
		t.Fatalf("expected keyword character-limit error, got %v", err)
	}
	if client.getCalls != 0 {
		t.Fatalf("expected no fetch calls, got %d", client.getCalls)
	}
	if len(client.createCalls) != 0 {
		t.Fatalf("expected no create calls, got %d", len(client.createCalls))
	}
	if len(client.updateCalls) != 0 {
		t.Fatalf("expected no update calls, got %d", len(client.updateCalls))
	}
}

func TestUploadVersionLocalizationsWithWarnings_DryRunWarnsOnlyForCreates(t *testing.T) {
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
	}

	valuesByLocale := map[string]map[string]string{
		"en-US": {
			"description": "Updated description",
			"keywords":    "updated",
			"supportUrl":  "https://example.com/en",
		},
		"ja": {
			"description": "日本語説明",
		},
	}

	results, warnings, err := UploadVersionLocalizationsWithWarnings(context.Background(), client, "version-id", valuesByLocale, true, SubmitReadinessOptions{})
	if err != nil {
		t.Fatalf("UploadVersionLocalizationsWithWarnings() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d (%+v)", len(warnings), warnings)
	}
	if warnings[0].Locale != "ja" {
		t.Fatalf("expected warning for ja locale, got %+v", warnings[0])
	}
	if warnings[0].Mode != SubmitReadinessCreateModePlanned {
		t.Fatalf("expected planned warning mode, got %+v", warnings[0])
	}
	if got := strings.Join(warnings[0].MissingFields, ","); got != "keywords,supportUrl" {
		t.Fatalf("unexpected missing fields %q", got)
	}
	if len(client.createCalls) != 0 {
		t.Fatalf("expected dry-run to avoid create calls, got %d", len(client.createCalls))
	}
	if len(client.updateCalls) != 0 {
		t.Fatalf("expected dry-run to avoid update calls, got %d", len(client.updateCalls))
	}
}

func TestUploadVersionLocalizationsWithWarnings_AppliedCompleteCreateDoesNotWarn(t *testing.T) {
	client := &stubVersionLocalizationClient{
		getResp: &asc.AppStoreVersionLocalizationsResponse{
			Data: []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{},
		},
	}

	valuesByLocale := map[string]map[string]string{
		"ja": {
			"description": "日本語説明",
			"keywords":    "一,二",
			"supportUrl":  "https://example.com/ja",
		},
	}

	results, warnings, err := UploadVersionLocalizationsWithWarnings(context.Background(), client, "version-id", valuesByLocale, false, SubmitReadinessOptions{})
	if err != nil {
		t.Fatalf("UploadVersionLocalizationsWithWarnings() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", warnings)
	}
	if len(client.createCalls) != 1 {
		t.Fatalf("expected one create call, got %d", len(client.createCalls))
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

func TestUploadVersionLocalizationsWithWarnings_RequiresWhatsNewWhenConfigured(t *testing.T) {
	client := &stubVersionLocalizationClient{
		getResp: &asc.AppStoreVersionLocalizationsResponse{
			Data: []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{},
		},
	}

	valuesByLocale := map[string]map[string]string{
		"ja": {
			"description": "日本語説明",
			"keywords":    "一,二",
			"supportUrl":  "https://example.com/ja",
		},
	}

	results, warnings, err := UploadVersionLocalizationsWithWarnings(
		context.Background(),
		client,
		"version-id",
		valuesByLocale,
		true,
		SubmitReadinessOptions{RequireWhatsNew: true},
	)
	if err != nil {
		t.Fatalf("UploadVersionLocalizationsWithWarnings() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %+v", warnings)
	}
	if warnings[0].Locale != "ja" {
		t.Fatalf("expected warning for ja locale, got %+v", warnings[0])
	}
	if len(warnings[0].MissingFields) != 1 || warnings[0].MissingFields[0] != "whatsNew" {
		t.Fatalf("expected missing fields [whatsNew], got %+v", warnings[0].MissingFields)
	}
}
