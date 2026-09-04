package shared

import "strings"

// certificateTypeValues mirrors the CertificateType enum in
// docs/openapi/latest.json. App Store Connect rejects any other value, so the
// CLI validates against this list before performing side effects such as
// generating a private key or issuing a create request.
var certificateTypeValues = []string{
	"APPLE_PAY",
	"APPLE_PAY_MERCHANT_IDENTITY",
	"APPLE_PAY_PSP_IDENTITY",
	"APPLE_PAY_RSA",
	"DEVELOPER_ID_APPLICATION",
	"DEVELOPER_ID_APPLICATION_G2",
	"DEVELOPER_ID_KEXT",
	"DEVELOPER_ID_KEXT_G2",
	"DEVELOPMENT",
	"DISTRIBUTION",
	"IDENTITY_ACCESS",
	"IOS_DEVELOPMENT",
	"IOS_DISTRIBUTION",
	"MAC_APP_DEVELOPMENT",
	"MAC_APP_DISTRIBUTION",
	"MAC_INSTALLER_DISTRIBUTION",
	"PASS_TYPE_ID",
	"PASS_TYPE_ID_WITH_NFC",
}

var certificateTypeSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(certificateTypeValues))
	for _, value := range certificateTypeValues {
		set[value] = struct{}{}
	}
	return set
}()

// CanonicalCertificateType normalizes value and returns the matching App Store
// Connect certificate type. Callers must use the returned value rather than the
// raw flag input: App Store Connect matches the enum exactly, so a normalized
// spelling such as "ios-distribution" has to reach the API as "IOS_DISTRIBUTION".
func CanonicalCertificateType(value string) (string, bool) {
	normalized := NormalizeEnumToken(value)
	if _, ok := certificateTypeSet[normalized]; !ok {
		return "", false
	}
	return normalized, true
}

// applePayCertificateTypePrefix marks the certificate types Apple issues against
// a merchant ID. They are valid CertificateType values, so list and filter paths
// accept them, but creating one requires a merchantId relationship the CLI does
// not send yet.
const applePayCertificateTypePrefix = "APPLE_PAY"

// isApplePayCertificateType reports whether the canonical certificate type is one
// of the Apple Pay types.
func isApplePayCertificateType(certificateType string) bool {
	return strings.HasPrefix(certificateType, applePayCertificateTypePrefix)
}

// CertificateCreateTypeList returns the certificate types asc certificates create
// can actually create: the full enum minus the Apple Pay types. Use it for create
// help text and diagnostics so both discovery paths agree with what the command
// accepts. Read paths such as certificates list stay unfiltered and keep
// accepting every enum value.
func CertificateCreateTypeList() []string {
	values := make([]string, 0, len(certificateTypeValues))
	for _, value := range certificateTypeValues {
		if isApplePayCertificateType(value) {
			continue
		}
		values = append(values, value)
	}
	return values
}

// ValidateCertificateCreateType returns the canonical certificate type for value,
// or a usage-class error when the certificate cannot be created by this CLI.
func ValidateCertificateCreateType(flagName, value string) (string, error) {
	canonical, ok := CanonicalCertificateType(value)
	if !ok {
		return "", UsageErrorf(
			"%s must be one of: %s (got %q)",
			flagName,
			strings.Join(CertificateCreateTypeList(), ", "),
			strings.TrimSpace(value),
		)
	}
	if isApplePayCertificateType(canonical) {
		return "", UsageErrorf(
			"%s %s needs a merchant ID relationship that asc certificates create does not support yet; inspect existing Apple Pay certificates with 'asc merchant-ids certificates list --merchant-id MERCHANT_ID' and create new ones in the Apple Developer portal",
			flagName,
			canonical,
		)
	}
	return canonical, nil
}
