//go:build darwin

package secureopen

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestRenameNoReplaceUnsupportedDarwin(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "filesystem unsupported", err: unix.ENOTSUP, want: true},
		{name: "syscall unavailable", err: unix.ENOSYS, want: true},
		{name: "invalid arguments", err: unix.EINVAL, want: false},
		{name: "destination exists", err: unix.EEXIST, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := renameNoReplaceUnsupportedDarwin(test.err); got != test.want {
				t.Fatalf("renameNoReplaceUnsupportedDarwin(%v) = %t, want %t", test.err, got, test.want)
			}
		})
	}
}
