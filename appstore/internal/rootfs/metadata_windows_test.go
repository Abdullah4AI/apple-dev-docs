//go:build windows

package rootfs

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestDACLInformationPreservesProtection(t *testing.T) {
	tests := []struct {
		name    string
		control windows.SECURITY_DESCRIPTOR_CONTROL
		want    windows.SECURITY_INFORMATION
	}{
		{
			name:    "protected",
			control: windows.SE_DACL_PROTECTED,
			want:    windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION,
		},
		{
			name: "inherited",
			want: windows.DACL_SECURITY_INFORMATION | windows.UNPROTECTED_DACL_SECURITY_INFORMATION,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := daclSecurityInformation(test.control); got != test.want {
				t.Fatalf("daclSecurityInformation() = %#x, want %#x", got, test.want)
			}
		})
	}
}
