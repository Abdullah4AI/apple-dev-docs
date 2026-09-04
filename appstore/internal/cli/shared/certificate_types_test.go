package shared

import (
	"slices"
	"strings"
	"testing"
)

func TestCanonicalCertificateTypeNormalizesSeparatorsAndCase(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "canonical", value: "IOS_DISTRIBUTION", want: "IOS_DISTRIBUTION"},
		{name: "lowercase", value: "ios_distribution", want: "IOS_DISTRIBUTION"},
		{name: "hyphenated", value: "ios-distribution", want: "IOS_DISTRIBUTION"},
		{name: "spaced", value: " mac installer distribution ", want: "MAC_INSTALLER_DISTRIBUTION"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CanonicalCertificateType(tt.value)
			if !ok {
				t.Fatalf("CanonicalCertificateType(%q) reported an unsupported type", tt.value)
			}
			if got != tt.want {
				t.Fatalf("CanonicalCertificateType(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestCanonicalCertificateTypeRejectsUnknownValues(t *testing.T) {
	for _, value := range []string{"", "DEVELOPER_ID_INSTALLER", "TVOS_DISTRIBUTION", "ios distribution 2"} {
		if got, ok := CanonicalCertificateType(value); ok {
			t.Fatalf("CanonicalCertificateType(%q) = %q, want rejection", value, got)
		}
	}
}

func TestCertificateCreateTypeListExcludesApplePay(t *testing.T) {
	values := CertificateCreateTypeList()
	if len(values) == 0 {
		t.Fatal("CertificateCreateTypeList() returned no values")
	}
	for _, value := range values {
		if strings.HasPrefix(value, "APPLE_PAY") {
			t.Fatalf("CertificateCreateTypeList() offers %q, which asc certificates create cannot create", value)
		}
	}
	if len(values) != len(certificateTypeValues)-4 {
		t.Fatalf("expected the four Apple Pay types to be the only exclusions, got %d of %d", len(values), len(certificateTypeValues))
	}
	if !slices.Contains(values, "MAC_INSTALLER_DISTRIBUTION") {
		t.Fatal("CertificateCreateTypeList() dropped a creatable type")
	}
}

func TestValidateCertificateCreateTypeReturnsCanonicalValue(t *testing.T) {
	got, err := ValidateCertificateCreateType("--certificate-type", "developer-id-application-g2")
	if err != nil {
		t.Fatalf("ValidateCertificateCreateType() error: %v", err)
	}
	if got != "DEVELOPER_ID_APPLICATION_G2" {
		t.Fatalf("ValidateCertificateCreateType() = %q, want %q", got, "DEVELOPER_ID_APPLICATION_G2")
	}
}

func TestValidateCertificateCreateTypeRejectsApplePayWithMerchantGuidance(t *testing.T) {
	for _, value := range []string{"APPLE_PAY", "apple-pay-merchant-identity", "APPLE_PAY_PSP_IDENTITY", "APPLE_PAY_RSA"} {
		_, err := ValidateCertificateCreateType("--certificate-type", value)
		if err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
		if !strings.Contains(err.Error(), "merchant ID relationship") {
			t.Fatalf("expected merchant ID guidance for %q, got %v", value, err)
		}
	}
}

func TestValidateCertificateCreateTypeDoesNotOfferApplePayForUnknownValues(t *testing.T) {
	_, err := ValidateCertificateCreateType("--certificate-type", "DEVELOPER_ID_INSTALLER")
	if err == nil {
		t.Fatal("expected an error for an unsupported certificate type")
	}
	if strings.Contains(err.Error(), "APPLE_PAY") {
		t.Fatalf("the allowed-values diagnostic must not recommend Apple Pay types, got %v", err)
	}
	if !strings.Contains(err.Error(), "MAC_INSTALLER_DISTRIBUTION") {
		t.Fatalf("expected the creatable types in the diagnostic, got %v", err)
	}
}
