//go:build darwin && !cgo

package signing

import (
	"strings"
	"testing"
)

func TestSigningRunSecurityFrameworkRequiresCGO(t *testing.T) {
	for _, err := range []error{
		createKeychainWithSecurityFramework("unused", nil),
		importPKCS12WithSecurityFramework("unused", nil, nil),
	} {
		if err == nil || !strings.Contains(err.Error(), "requires a cgo-enabled macOS build") {
			t.Fatalf("error = %v, want cgo requirement", err)
		}
	}
}
