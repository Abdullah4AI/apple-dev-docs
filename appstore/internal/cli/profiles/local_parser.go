package profiles

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"go.mozilla.org/pkcs7"
	"howett.net/plist"
)

type mobileProvision struct {
	UUID           string         `plist:"UUID"`
	Name           string         `plist:"Name"`
	TeamIdentifier []string       `plist:"TeamIdentifier"`
	CreationDate   time.Time      `plist:"CreationDate"`
	ExpirationDate time.Time      `plist:"ExpirationDate"`
	Entitlements   map[string]any `plist:"Entitlements"`
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
