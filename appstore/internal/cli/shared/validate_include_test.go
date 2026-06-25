package shared

import "testing"

func TestValidateInclude(t *testing.T) {
	allowed := []string{"build", "tester"}

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty is valid", value: "", wantErr: false},
		{name: "single allowed", value: "build", wantErr: false},
		{name: "multiple allowed", value: "build,tester", wantErr: false},
		{name: "whitespace tolerated", value: " build , tester ", wantErr: false},
		{name: "unknown value", value: "crashLog", wantErr: true},
		{name: "mixed valid and invalid", value: "build,bogus", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInclude(tc.value, allowed...)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tc.value)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.value, err)
			}
		})
	}
}
