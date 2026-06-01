package appleads

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/auth"
)

const (
	tokenURL         = "https://appleid.apple.com/auth/oauth2/token"
	tokenScope       = "searchadsorg"
	clientSecretTTL  = 10 * time.Minute
	tokenRefreshSkew = 30 * time.Second
	clientSecretAud  = "https://appleid.apple.com"
	grantClientCreds = "client_credentials"
)

// Credentials contains resolved Apple Ads authentication inputs.
type Credentials struct {
	ClientID       string
	TeamID         string
	KeyID          string
	PrivateKeyPath string
	PrivateKeyPEM  string
	AccessToken    string
	OrgID          string
	Profile        string
}

type tokenCache struct {
	accessToken string
	expiresAt   time.Time
}

func (c *Client) bearerToken(ctx context.Context) (string, error) {
	if strings.TrimSpace(c.credentials.AccessToken) != "" {
		return strings.TrimSpace(c.credentials.AccessToken), nil
	}

	now := c.now()
	c.tokenMu.Lock()
	if c.token.accessToken != "" && now.Before(c.token.expiresAt.Add(-tokenRefreshSkew)) {
		token := c.token.accessToken
		c.tokenMu.Unlock()
		return token, nil
	}
	c.tokenMu.Unlock()

	privateKey, err := c.privateKey()
	if err != nil {
		return "", err
	}
	clientSecret, err := GenerateClientSecret(c.credentials.KeyID, c.credentials.TeamID, c.credentials.ClientID, privateKey, now, clientSecretTTL)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", grantClientCreds)
	form.Set("client_id", c.credentials.ClientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", tokenScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", parseError(body, resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("token response missing access_token")
	}
	if !strings.EqualFold(tokenResp.TokenType, "Bearer") {
		return "", fmt.Errorf("token response has unsupported token_type %q", tokenResp.TokenType)
	}
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}

	c.tokenMu.Lock()
	c.token = tokenCache{
		accessToken: tokenResp.AccessToken,
		expiresAt:   now.Add(time.Duration(expiresIn) * time.Second),
	}
	c.tokenMu.Unlock()

	return tokenResp.AccessToken, nil
}

// AccessToken returns a bearer token for diagnostics and auth token commands.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	return c.bearerToken(ctx)
}

func (c *Client) privateKey() (*ecdsa.PrivateKey, error) {
	c.privateKeyMu.Lock()
	defer c.privateKeyMu.Unlock()

	if c.privateKeyValue != nil {
		return c.privateKeyValue, nil
	}

	var (
		key *ecdsa.PrivateKey
		err error
	)
	if strings.TrimSpace(c.credentials.PrivateKeyPEM) != "" {
		key, err = auth.LoadPrivateKeyFromPEM([]byte(c.credentials.PrivateKeyPEM))
	} else {
		key, err = auth.LoadPrivateKey(c.credentials.PrivateKeyPath)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	c.privateKeyValue = key
	return key, nil
}

// GenerateClientSecret creates an Apple Ads OAuth client secret JWT.
func GenerateClientSecret(keyID, teamID, clientID string, privateKey *ecdsa.PrivateKey, now time.Time, lifetime time.Duration) (string, error) {
	if lifetime <= 0 {
		lifetime = clientSecretTTL
	}
	if lifetime > 180*24*time.Hour {
		return "", fmt.Errorf("client secret lifetime must be <= 180 days")
	}
	claims := jwt.MapClaims{
		"iss": teamID,
		"iat": jwt.NewNumericDate(now),
		"exp": jwt.NewNumericDate(now.Add(lifetime)),
		"aud": clientSecretAud,
		"sub": clientID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign client secret: %w", err)
	}
	return signed, nil
}
