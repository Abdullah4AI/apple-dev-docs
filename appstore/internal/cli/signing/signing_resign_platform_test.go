package signing

import (
	"strings"
	"testing"
)

func TestSigningResignPlatformStringsRequireLosslessArrayShape(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   any
		wantLen int
		wantErr string
	}{
		{name: "single string", value: []string{"iPhoneOS"}, wantLen: 1},
		{name: "multiple strings", value: []any{"iPhoneOS", "WatchOS"}, wantLen: 2},
		{name: "non-string member", value: []any{"iPhoneOS", 42}, wantErr: "non-empty string"},
		{name: "scalar", value: "iPhoneOS", wantErr: "array of strings"},
		{name: "object", value: map[string]any{"platform": "iPhoneOS"}, wantErr: "array of strings"},
		{name: "empty array", value: []any{}, wantErr: "at least one"},
		{name: "empty member", value: []any{""}, wantErr: "non-empty string"},
		{name: "control member", value: []any{"iPhoneOS\n"}, wantErr: "non-empty string"},
		{name: "duplicate member", value: []any{"iPhoneOS", "iPhoneOS"}, wantErr: "duplicate"},
		{name: "case duplicate", value: []any{"iPhoneOS", "iphoneos"}, wantErr: "duplicate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := signingResignPlatformStrings(test.value)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("signingResignPlatformStrings() error = %v", err)
				}
				if len(got) != test.wantLen {
					t.Fatalf("signingResignPlatformStrings() length = %d, want %d", len(got), test.wantLen)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("signingResignPlatformStrings() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestSigningResignPlatformRejectsMalformedMetadataAsOperationalError(t *testing.T) {
	for _, value := range []any{
		[]any{"iPhoneOS", 42},
		"iPhoneOS",
		map[string]any{"platform": "iPhoneOS"},
		[]any{""},
		[]any{"iPhoneOS\u202e"},
	} {
		err := validateSigningResignPlatform(map[string]any{
			"DTPlatformName":             "iphoneos",
			"CFBundleSupportedPlatforms": value,
		}, "application")
		if err == nil {
			t.Fatalf("validateSigningResignPlatform(%#v) returned nil", value)
		}
		if isSigningResignUsageError(err) {
			t.Fatalf("validateSigningResignPlatform(%#v) classified malformed metadata as usage: %v", value, err)
		}
	}
}

func TestSigningResignPlatformAppliesExactCanonicalTargetPolicy(t *testing.T) {
	for _, test := range []struct {
		name string
		kind string
		info map[string]any
	}{
		{
			name: "iOS canonical",
			kind: "application",
			info: map[string]any{
				"DTPlatformName":             "iphoneos",
				"CFBundleSupportedPlatforms": []string{"iPhoneOS"},
			},
		},
		{
			name: "watch canonical",
			kind: "watch-application",
			info: map[string]any{
				"DTPlatformName":             "watchos",
				"CFBundleSupportedPlatforms": []string{"WatchOS"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateSigningResignPlatform(test.info, test.kind); err != nil {
				t.Fatalf("validateSigningResignPlatform() error = %v", err)
			}
		})
	}
}
