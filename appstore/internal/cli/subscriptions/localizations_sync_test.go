package subscriptions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLocalizationSyncEntriesRejectsStrictInputErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "unknown field", body: `{"en-US":{"name":"Pro","unexpected":"value"}}`, want: `unknown field "unexpected"`},
		{name: "null field", body: `{"en-US":{"name":null}}`, want: `must be a string, not null`},
		{name: "duplicate field", body: `{"en-US":{"name":"One","name":"Two"}}`, want: `duplicate field "name"`},
		{name: "trailing document", body: `{"en-US":{"name":"Pro"}} {}`, want: `invalid JSON`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("write input: %v", err)
			}
			_, err := readLocalizationSyncEntries(path, "description")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestExecuteLocalizationSyncRejectsDuplicateRemoteLocaleBeforeMutation(t *testing.T) {
	mutations := 0
	api := localizationSyncAPI{
		Type: "subscriptionLocalizations",
		List: func(context.Context) ([]localizationSyncRemote, error) {
			return []localizationSyncRemote{
				{ID: "loc-1", Locale: "en_US", Fields: map[string]string{"name": "One"}},
				{ID: "loc-2", Locale: "en-US", Fields: map[string]string{"name": "Two"}},
			}, nil
		},
		Create: func(context.Context, localizationSyncEntry) (localizationSyncRemote, error) {
			mutations++
			return localizationSyncRemote{}, nil
		},
		Update: func(context.Context, localizationSyncRemote, localizationSyncEntry) (localizationSyncRemote, error) {
			mutations++
			return localizationSyncRemote{}, nil
		},
	}

	_, err := executeLocalizationSync(
		context.Background(),
		api,
		"sub-1",
		"input.json",
		[]localizationSyncEntry{{Locale: "en-US", Fields: map[string]string{"name": "Desired"}}},
		false,
		"subscriptions localizations sync",
	)
	if err == nil || !strings.Contains(err.Error(), `duplicate remote locale "en-US"`) {
		t.Fatalf("expected duplicate remote locale error, got %v", err)
	}
	if mutations != 0 {
		t.Fatalf("expected no mutations, got %d", mutations)
	}
}

func TestExecuteLocalizationSyncCancellationArtifactsRemainingLocales(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	mutations := 0
	api := localizationSyncAPI{
		Type: "subscriptionLocalizations",
		List: func(context.Context) ([]localizationSyncRemote, error) {
			return nil, nil
		},
		Create: func(context.Context, localizationSyncEntry) (localizationSyncRemote, error) {
			mutations++
			cancel()
			return localizationSyncRemote{}, context.Canceled
		},
		Update: func(context.Context, localizationSyncRemote, localizationSyncEntry) (localizationSyncRemote, error) {
			t.Fatal("unexpected update")
			return localizationSyncRemote{}, nil
		},
	}
	entries := []localizationSyncEntry{
		{Locale: "de-DE", Fields: map[string]string{"name": "Deutsch"}},
		{Locale: "en-US", Fields: map[string]string{"name": "English"}},
		{Locale: "fr-FR", Fields: map[string]string{"name": "Francais"}},
	}

	summary, err := executeLocalizationSync(ctx, api, "sub-1", "input.json", entries, false, "subscriptions localizations sync")
	if err == nil || summary == nil || summary.Failed != 3 || mutations != 1 {
		t.Fatalf("unexpected cancellation result: summary=%+v mutations=%d err=%v", summary, mutations, err)
	}
	if summary.FailureArtifactPath == "" || len(summary.Results) != 3 {
		t.Fatalf("expected all remaining locales in artifact summary: %+v", summary)
	}
	payload, readErr := os.ReadFile(summary.FailureArtifactPath)
	if readErr != nil {
		t.Fatalf("read artifact: %v", readErr)
	}
	var artifact localizationSyncFailureArtifact
	if err := json.Unmarshal(payload, &artifact); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	if len(artifact.RetryInput) != 3 {
		t.Fatalf("expected three replay locales, got %+v", artifact.RetryInput)
	}
}
