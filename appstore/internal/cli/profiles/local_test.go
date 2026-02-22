package profiles

import (
	"testing"
	"time"
)

func TestIsExpired(t *testing.T) {
	t0 := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		now       time.Time
		want      bool
	}{
		{name: "zero", expiresAt: time.Time{}, now: t0, want: false},
		{name: "before", expiresAt: t0.Add(1 * time.Second), now: t0, want: false},
		{name: "equal", expiresAt: t0, now: t0, want: true},
		{name: "after", expiresAt: t0.Add(-1 * time.Second), now: t0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExpired(tt.expiresAt, tt.now); got != tt.want {
				t.Fatalf("isExpired(expiresAt=%s, now=%s)=%t, want %t", tt.expiresAt.Format(time.RFC3339Nano), tt.now.Format(time.RFC3339Nano), got, tt.want)
			}
		})
	}
}
