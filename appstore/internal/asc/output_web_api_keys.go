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
