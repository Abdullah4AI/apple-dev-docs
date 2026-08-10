package capabilities

import (
	"strings"
	"testing"
	"unicode"
)

func TestAssetLibraryIsNotInPublicCapabilities(t *testing.T) {
	for _, capability := range capabilityRows() {
		fields := []string{
			capability.Area,
			capability.Capability,
			strings.Join(capability.Commands, " "),
			strings.Join(capability.APIResources, " "),
			strings.Join(capability.Notes, " "),
			capability.NextAction,
		}
		row := strings.ToLower(strings.Join(fields, " "))
		if containsAssetLibraryTerm(row) {
			t.Fatalf("Asset Library must remain absent from public capabilities, got %+v", capability)
		}
	}
}

func TestContainsAssetLibraryTerm(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "asset library", want: true},
		{value: "asc asset-library", want: true},
		{value: "creative-assets", want: true},
		{value: "assetLibraryItems", want: true},
		{value: "asset catalog", want: false},
		{value: "creative tools", want: false},
	}

	for _, test := range tests {
		if got := containsAssetLibraryTerm(test.value); got != test.want {
			t.Errorf("containsAssetLibraryTerm(%q) = %t, want %t", test.value, got, test.want)
		}
	}
}

func containsAssetLibraryTerm(value string) bool {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
	return strings.Contains(normalized, "assetlibrary") || strings.Contains(normalized, "creativeasset")
}
