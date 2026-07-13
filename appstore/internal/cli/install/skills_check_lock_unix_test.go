//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package install

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRetrySkillsCheckFcntlRetriesInterruptedCalls(t *testing.T) {
	original := fcntlSkillsCheckClaim
	t.Cleanup(func() { fcntlSkillsCheckClaim = original })

	calls := 0
	fcntlSkillsCheckClaim = func(_ uintptr, _ int, _ *unix.Flock_t) error {
		calls++
		if calls < 3 {
			return unix.EINTR
		}
		return nil
	}

	if err := retrySkillsCheckFcntl(0, unix.F_SETLKW, &unix.Flock_t{}); err != nil {
		t.Fatalf("retrySkillsCheckFcntl() error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("fcntl calls = %d, want 3", calls)
	}
}

func TestRetrySkillsCheckFcntlReturnsNonInterruptedError(t *testing.T) {
	original := fcntlSkillsCheckClaim
	t.Cleanup(func() { fcntlSkillsCheckClaim = original })

	want := errors.New("lock failed")
	fcntlSkillsCheckClaim = func(_ uintptr, _ int, _ *unix.Flock_t) error {
		return want
	}

	if err := retrySkillsCheckFcntl(0, unix.F_SETLK, &unix.Flock_t{}); !errors.Is(err, want) {
		t.Fatalf("retrySkillsCheckFcntl() error = %v, want %v", err, want)
	}
}
