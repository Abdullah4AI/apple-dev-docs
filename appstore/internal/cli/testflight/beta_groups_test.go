package testflight

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		done <- buf.String()
	}()

	defer func() {
		os.Stderr = oldStderr
	}()

	fn()

	_ = w.Close()
	return <-done
}

func makeBetaGroupsFixture(internalCount, externalCount int) *asc.BetaGroupsResponse {
	data := make([]asc.Resource[asc.BetaGroupAttributes], 0, internalCount+externalCount)
	for i := range internalCount {
		data = append(data, asc.Resource[asc.BetaGroupAttributes]{
			Type:       asc.ResourceTypeBetaGroups,
			ID:         fmt.Sprintf("internal-%d", i),
			Attributes: asc.BetaGroupAttributes{Name: fmt.Sprintf("Internal %d", i), IsInternalGroup: true},
		})
	}
	for i := range externalCount {
		data = append(data, asc.Resource[asc.BetaGroupAttributes]{
			Type:       asc.ResourceTypeBetaGroups,
			ID:         fmt.Sprintf("external-%d", i),
			Attributes: asc.BetaGroupAttributes{Name: fmt.Sprintf("External %d", i), IsInternalGroup: false},
		})
	}
	return &asc.BetaGroupsResponse{Data: data}
}

func TestFilterBetaGroupsByInternal_WarnsWhenLimitTruncates(t *testing.T) {
	groups := makeBetaGroupsFixture(5, 2)

	var filtered asc.BetaGroupsResponse
	stderr := captureStderr(t, func() {
		filtered = filterBetaGroupsByInternal(groups, true, 2)
	})

	if len(filtered.Data) != 2 {
		t.Fatalf("expected 2 groups after truncation, got %d", len(filtered.Data))
	}
	for _, g := range filtered.Data {
		if !g.Attributes.IsInternalGroup {
			t.Fatalf("expected only internal groups, got %q", g.ID)
		}
	}
	want := "Warning: showing 2 of 5 filtered groups (--limit 2); rerun without --limit for all\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestFilterBetaGroupsByInternal_NoWarningWithoutTruncation(t *testing.T) {
	groups := makeBetaGroupsFixture(2, 3)

	var filtered asc.BetaGroupsResponse
	stderr := captureStderr(t, func() {
		filtered = filterBetaGroupsByInternal(groups, true, 5)
	})

	if len(filtered.Data) != 2 {
		t.Fatalf("expected 2 internal groups, got %d", len(filtered.Data))
	}
	if stderr != "" {
		t.Fatalf("expected no warning when limit is not exceeded, got %q", stderr)
	}
}

func TestFilterBetaGroupsByInternal_NoWarningWithoutLimit(t *testing.T) {
	groups := makeBetaGroupsFixture(1, 4)

	var filtered asc.BetaGroupsResponse
	stderr := captureStderr(t, func() {
		filtered = filterBetaGroupsByInternal(groups, false, 0)
	})

	if len(filtered.Data) != 4 {
		t.Fatalf("expected 4 external groups, got %d", len(filtered.Data))
	}
	for _, g := range filtered.Data {
		if g.Attributes.IsInternalGroup {
			t.Fatalf("expected only external groups, got %q", g.ID)
		}
	}
	if stderr != "" {
		t.Fatalf("expected no warning without --limit, got %q", stderr)
	}
}
