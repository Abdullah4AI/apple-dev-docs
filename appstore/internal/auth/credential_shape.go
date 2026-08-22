package auth

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// keyIDMinLength and keyIDMaxLength bracket the App Store Connect API key ID
// shape. Apple issues 10-character identifiers such as 39MX87M9Y4; the wider
// range keeps the heuristic conservative because it is only ever used to
// strengthen a hint, never to reject a value.
const (
	keyIDMinLength = 8
	keyIDMaxLength = 12
)

// CredentialShapeLabels names the fields being inspected so findings speak in
// the caller's vocabulary: environment variables for `auth doctor`, flags for
// `auth login`.
type CredentialShapeLabels struct {
	KeyID    string
	IssuerID string
}

// CredentialShapeFinding is a non-blocking observation about the shape of a key
// ID or issuer ID. Messages never include the inspected values so reports stay
// safe to paste into a bug report or CI log.
type CredentialShapeFinding struct {
	Field          string
	Message        string
	Recommendation string
	// DefiniteSwap marks the only unambiguous case: the key ID is an issuer
	// UUID and the issuer ID also resembles an API key ID.
	DefiniteSwap bool
}

// LooksLikeIssuerID reports whether value has the App Store Connect issuer ID
// shape: a canonical hyphenated UUID such as
// 69a6de00-aaaa-bbbb-cccc-123456789abc. Non-canonical spellings accepted by
// uuid.Parse (braced, URN, unhyphenated) are rejected because Apple only ever
// issues the canonical form.
func LooksLikeIssuerID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return false
	}
	return strings.EqualFold(trimmed, parsed.String())
}

// LooksLikeKeyID reports whether value has the App Store Connect API key ID
// shape: a short uppercase alphanumeric identifier such as 39MX87M9Y4. A false
// result never means the value is invalid; it only means the value carries no
// evidence of being a key ID.
func LooksLikeKeyID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < keyIDMinLength || len(trimmed) > keyIDMaxLength {
		return false
	}
	for _, r := range trimmed {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// InspectCredentialShapes reports key ID and issuer ID values whose shapes
// suggest the two were swapped, or an issuer ID that cannot be a UUID.
//
// Only the issuer ID has a format Apple guarantees, so key IDs are never
// format-policed: an unusual but plausible key ID (all digits, mixed case,
// shorter or longer than 10 characters) produces no finding. The key ID is
// flagged only when it is itself a canonical UUID, which is the shape of the
// other field.
func InspectCredentialShapes(labels CredentialShapeLabels, keyID, issuerID string) []CredentialShapeFinding {
	keyID = strings.TrimSpace(keyID)
	issuerID = strings.TrimSpace(issuerID)

	keyLooksLikeIssuer := LooksLikeIssuerID(keyID)
	issuerIsMalformed := issuerID != "" && !LooksLikeIssuerID(issuerID)
	issuerLooksLikeKey := issuerIsMalformed && LooksLikeKeyID(issuerID)
	swapHintSupported := keyLooksLikeIssuer && issuerLooksLikeKey
	definiteSwap := swapHintSupported
	swapSuffix := ""
	if swapHintSupported {
		swapSuffix = " — the values may be swapped"
	}

	recommendation := fmt.Sprintf(
		"Set %s to the App Store Connect API key ID and %s to the issuer UUID",
		labels.KeyID,
		labels.IssuerID,
	)

	var findings []CredentialShapeFinding
	if keyLooksLikeIssuer {
		findings = append(findings, CredentialShapeFinding{
			Field:          labels.KeyID,
			Message:        fmt.Sprintf("%s looks like an issuer ID%s", labels.KeyID, swapSuffix),
			Recommendation: recommendation,
			DefiniteSwap:   definiteSwap,
		})
	}
	if issuerIsMalformed {
		message := fmt.Sprintf("%s is not a UUID", labels.IssuerID)
		switch {
		case issuerLooksLikeKey:
			message = fmt.Sprintf("%s looks like a key ID%s", labels.IssuerID, swapSuffix)
		}
		findings = append(findings, CredentialShapeFinding{
			Field:          labels.IssuerID,
			Message:        message,
			Recommendation: recommendation,
			DefiniteSwap:   definiteSwap,
		})
	}
	return findings
}
