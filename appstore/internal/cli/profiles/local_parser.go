package profiles

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.mozilla.org/pkcs7"
	"howett.net/plist"
)

type mobileProvision struct {
	UUID                  string         `plist:"UUID"`
	Name                  string         `plist:"Name"`
	AppIDName             string         `plist:"AppIDName"`
	TeamName              string         `plist:"TeamName"`
	TeamIdentifier        []string       `plist:"TeamIdentifier"`
	Platform              []string       `plist:"Platform"`
	ProvisionedDevices    []string       `plist:"ProvisionedDevices"`
	ProvisionsAllDevices  bool           `plist:"ProvisionsAllDevices"`
	CreationDate          time.Time      `plist:"CreationDate"`
	ExpirationDate        time.Time      `plist:"ExpirationDate"`
	TimeToLive            int            `plist:"TimeToLive"`
	Entitlements          map[string]any `plist:"Entitlements"`
	DeveloperCertificates [][]byte       `plist:"DeveloperCertificates"`
}

func parseMobileProvision(data []byte) (*mobileProvision, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("profile file is empty")
	}

	plistBytes := data
	if p7, err := pkcs7.Parse(data); err == nil && len(p7.Content) > 0 {
		plistBytes = p7.Content
	}

	var mp mobileProvision
	decoder := plist.NewDecoder(bytes.NewReader(plistBytes))
	if err := decoder.Decode(&mp); err != nil {
		return nil, fmt.Errorf("decode embedded plist: %w", err)
	}
	return &mp, nil
}

func (m *mobileProvision) TeamID() string {
	if m == nil {
		return ""
	}
	if len(m.TeamIdentifier) > 0 {
		return strings.TrimSpace(m.TeamIdentifier[0])
	}
	if v := strings.TrimSpace(coerceAnyToString(m.Entitlements["com.apple.developer.team-identifier"])); v != "" {
		return v
	}
	return ""
}

func (m *mobileProvision) ApplicationIdentifier() string {
	if m == nil {
		return ""
	}
	// Common key in mobileprovision profiles.
	if v := strings.TrimSpace(coerceAnyToString(m.Entitlements["application-identifier"])); v != "" {
		return v
	}
	// Best-effort fallback.
	return strings.TrimSpace(coerceAnyToString(m.Entitlements["com.apple.application-identifier"]))
}

func (m *mobileProvision) BundleID() string {
	if m == nil {
		return ""
	}
	appID := strings.TrimSpace(m.ApplicationIdentifier())
	if appID == "" {
		return ""
	}

	// Typical format: TEAMID.com.example.app or TEAMID.*
	if team := strings.TrimSpace(m.TeamID()); team != "" {
		prefix := team + "."
		if strings.HasPrefix(appID, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(appID, prefix))
		}
	}

	// Fallback: strip first component.
	if parts := strings.SplitN(appID, ".", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func coerceAnyToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return ""
	}
}

func inspectDeveloperCertificate(data []byte) profileCertificate {
	sumSHA1 := sha1.Sum(data)
	sumSHA256 := sha256.Sum256(data)

	cert := profileCertificate{
		SHA1:   strings.ToUpper(hex.EncodeToString(sumSHA1[:])),
		SHA256: strings.ToUpper(hex.EncodeToString(sumSHA256[:])),
	}
	parsed, err := x509.ParseCertificate(data)
	if err != nil {
		return cert
	}
	cert.CommonName = strings.TrimSpace(parsed.Subject.CommonName)
	cert.SerialNumber = strings.TrimSpace(parsed.SerialNumber.String())
	cert.NotBefore = parsed.NotBefore
	cert.NotAfter = parsed.NotAfter
	return cert
}
