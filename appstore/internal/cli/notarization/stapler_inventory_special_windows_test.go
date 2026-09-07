//go:build windows

package notarization

import "testing"

func makeStaplerSpecialEntry(t *testing.T, _ string) {
	t.Helper()
	t.Skip("named pipes are not portable on Windows")
}
