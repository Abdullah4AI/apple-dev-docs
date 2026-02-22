package subscriptions

import (
	"strings"
	"testing"
)

func TestNormalizeSubscriptionGracePeriodDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		required bool
		want     string
		wantErr  string
	}{
		{
			name:  "empty optional",
			input: "",
			want:  "",
		},
		{
			name:     "empty required",
			input:    " ",
			required: true,
			wantErr:  "--duration is required",
		},
		{
			name:  "spec value",
			input: "SIXTEEN_DAYS",
			want:  "SIXTEEN_DAYS",
		},
		{
			name:  "spec value lowercase",
			input: "sixteen_days",
			want:  "SIXTEEN_DAYS",
		},
		{
			name:  "legacy value",
			input: "DAY_16",
			want:  "DAY_16",
		},
		{
			name:  "legacy value lowercase",
			input: "day_16",
			want:  "DAY_16",
		},
		{
			name:    "invalid value",
			input:   "BAD",
			wantErr: "--duration must be one of",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeSubscriptionGracePeriodDuration(test.input, test.required)
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error %q, got %q", test.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}
