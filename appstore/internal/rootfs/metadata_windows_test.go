//go:build windows

package rootfs

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestShouldSetReplacementDACLPreservesNullAndProtectedLists(t *testing.T) {
	shouldSet, err := shouldSetReplacementDACL(0, nil)
	if err != nil {
		t.Fatalf("shouldSetReplacementDACL() error = %v", err)
	}
	if !shouldSet {
		t.Fatal("unprotected NULL DACL must be copied onto a replacement")
	}

	shouldSet, err = shouldSetReplacementDACL(windows.SE_DACL_PROTECTED, nil)
	if err != nil {
		t.Fatalf("shouldSetReplacementDACL(protected) error = %v", err)
	}
	if !shouldSet {
		t.Fatal("protected DACL must be copied even when empty")
	}

	shouldSet, err = shouldSetReplacementDACL(0, &windows.ACL{})
	if err != nil {
		t.Fatalf("shouldSetReplacementDACL(empty) error = %v", err)
	}
	if !shouldSet {
		t.Fatal("empty DACL denies all access and must be copied")
	}
}

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
