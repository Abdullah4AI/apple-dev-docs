package asc

import (
	"fmt"
	"strings"
)

// WebAPIKeyActor is a non-secret user or actor reference on an API key.
type WebAPIKeyActor struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// WebAPIKeyListItem is one team or individual API key from a web-session list.
type WebAPIKeyListItem struct {
	KeyID       string          `json:"keyId"`
	Name        string          `json:"name,omitempty"`
	Kind        string          `json:"kind"`
	Roles       []string        `json:"roles,omitempty"`
	Active      bool            `json:"active"`
	KeyType     string          `json:"keyType,omitempty"`
	LastUsed    string          `json:"lastUsed,omitempty"`
	GeneratedBy *WebAPIKeyActor `json:"generatedBy,omitempty"`
	RevokedBy   *WebAPIKeyActor `json:"revokedBy,omitempty"`
}

// WebAPIKeysListResult is the computed list of API keys visible to a web session.
type WebAPIKeysListResult struct {
	Keys []WebAPIKeyListItem `json:"keys"`
}

// WebAPIKeyGetResult is non-secret metadata for one team API key.
type WebAPIKeyGetResult struct {
	KeyID          string   `json:"keyId"`
	Name           string   `json:"name,omitempty"`
	IssuerID       string   `json:"issuerId,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	Active         bool     `json:"active"`
	AllAppsVisible bool     `json:"allAppsVisible"`
	CanDownload    bool     `json:"canDownload"`
	KeyType        string   `json:"keyType,omitempty"`
	LastUsed       string   `json:"lastUsed,omitempty"`
	RevokingDate   string   `json:"revokingDate,omitempty"`
}

// WebAPIKeyCreateIndividualResult is the non-secret receipt from creating an
// individual API key through a web session. The private and public key bytes
// are deliberately not part of this output contract.
type WebAPIKeyCreateIndividualResult struct {
	KeyID      string `json:"keyId"`
	UserID     string `json:"userId"`
	P8Path     string `json:"p8Path"`
	Active     bool   `json:"active"`
	Registered bool   `json:"registered"`
}

func webAPIKeyCreateIndividualRows(result *WebAPIKeyCreateIndividualResult) ([]string, [][]string) {
	headers := []string{"Key ID", "User ID", "Active", "Registered", "P8 Path"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.KeyID,
		result.UserID,
		fmt.Sprintf("%t", result.Active),
		fmt.Sprintf("%t", result.Registered),
		result.P8Path,
	}}
}

// WebAPIKeyRevokeResult is the receipt for a verified API-key revocation.
// Changed is false when the selected key was already inactive and no write was
// sent.
type WebAPIKeyRevokeResult struct {
	KeyID   string `json:"keyId"`
	Kind    string `json:"kind"`
	Changed bool   `json:"changed"`
	Active  bool   `json:"active"`
	Status  string `json:"status"`
}

func webAPIKeysListRows(result *WebAPIKeysListResult) ([]string, [][]string) {
	headers := []string{"Key ID", "Name", "Kind", "Roles", "Active"}
	if result == nil {
		return headers, nil
	}
	rows := make([][]string, 0, len(result.Keys))
	for _, item := range result.Keys {
		rows = append(rows, []string{
			item.KeyID,
			item.Name,
			item.Kind,
			strings.Join(item.Roles, ", "),
			fmt.Sprintf("%t", item.Active),
		})
	}
	return headers, rows
}

func webAPIKeyGetRows(result *WebAPIKeyGetResult) ([]string, [][]string) {
	if result == nil {
		result = &WebAPIKeyGetResult{}
	}
	return []string{"Key ID", "Name", "Issuer ID", "Roles", "Active"}, [][]string{{
		result.KeyID,
		result.Name,
		result.IssuerID,
		strings.Join(result.Roles, ", "),
		fmt.Sprintf("%t", result.Active),
	}}
}

func webAPIKeyRevokeRows(result *WebAPIKeyRevokeResult) ([]string, [][]string) {
	headers := []string{"Key ID", "Kind", "Changed", "Active", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.KeyID,
		result.Kind,
		fmt.Sprintf("%t", result.Changed),
		fmt.Sprintf("%t", result.Active),
		result.Status,
	}}
}
