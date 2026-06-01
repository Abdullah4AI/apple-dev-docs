package appleads

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

const BaseURL = "https://api.searchads.apple.com/api/"

// RawResponse preserves the Apple Ads response envelope.
type RawResponse json.RawMessage

// MarshalJSON implements json.Marshaler.
func (r RawResponse) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(r).MarshalJSON()
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// Client is an Apple Ads Campaign Management API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	tokenURL   string
	now        func() time.Time

	credentials Credentials

	tokenMu         sync.Mutex
	token           tokenCache
	privateKeyMu    sync.Mutex
	privateKeyValue *ecdsa.PrivateKey
}

// NewClient constructs an Apple Ads API client.
func NewClient(credentials Credentials, opts ...ClientOption) (*Client, error) {
	client := &Client{
		httpClient:  &http.Client{Timeout: asc.ResolveTimeout()},
		baseURL:     BaseURL,
		tokenURL:    tokenURL,
		now:         time.Now,
		credentials: normalizeCredentials(credentials),
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: asc.ResolveTimeout()}
	}
	if strings.TrimSpace(client.baseURL) == "" {
		client.baseURL = BaseURL
	}
	if strings.TrimSpace(client.tokenURL) == "" {
		client.tokenURL = tokenURL
	}
	if client.now == nil {
		client.now = time.Now
	}
	if err := validateCredentials(client.credentials); err != nil {
		return nil, err
	}
	return client, nil
}

func normalizeCredentials(credentials Credentials) Credentials {
	credentials.ClientID = strings.TrimSpace(credentials.ClientID)
	credentials.TeamID = strings.TrimSpace(credentials.TeamID)
	credentials.KeyID = strings.TrimSpace(credentials.KeyID)
	credentials.PrivateKeyPath = strings.TrimSpace(credentials.PrivateKeyPath)
	credentials.PrivateKeyPEM = strings.TrimSpace(credentials.PrivateKeyPEM)
	credentials.AccessToken = strings.TrimSpace(credentials.AccessToken)
	credentials.OrgID = strings.TrimSpace(credentials.OrgID)
	credentials.Profile = strings.TrimSpace(credentials.Profile)
	return credentials
}

func validateCredentials(credentials Credentials) error {
	if credentials.AccessToken != "" {
		return nil
	}
	if credentials.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if credentials.TeamID == "" {
		return fmt.Errorf("team ID is required")
	}
	if credentials.KeyID == "" {
		return fmt.Errorf("key ID is required")
	}
	if credentials.PrivateKeyPath == "" && credentials.PrivateKeyPEM == "" {
		return fmt.Errorf("private key is required")
	}
	return nil
}

// WithHTTPClient configures the HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

// WithBaseURL configures the Apple Ads API base URL.
func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) {
		client.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/"
	}
}

// WithTokenURL configures the OAuth token URL.
func WithTokenURL(tokenURL string) ClientOption {
	return func(client *Client) {
		client.tokenURL = strings.TrimSpace(tokenURL)
	}
}

// WithNow configures the clock used by token caching.
func WithNow(now func() time.Time) ClientOption {
	return func(client *Client) {
		client.now = now
	}
}

// Do executes a documented Apple Ads endpoint.
func (c *Client) Do(ctx context.Context, spec EndpointSpec, pathParams map[string]string, query url.Values, body json.RawMessage) (RawResponse, error) {
	path, err := expandPath(spec.Path, pathParams)
	if err != nil {
		return nil, err
	}
	return c.Request(ctx, spec.Method, path, query, body, spec.RequiresOrg)
}

// Request executes an Apple Ads API request for a relative v5 path.
func (c *Client) Request(ctx context.Context, method, path string, query url.Values, body json.RawMessage, requiresOrg bool) (RawResponse, error) {
	token, err := c.bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	requestURL, err := c.requestURL(path, query)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if requiresOrg {
		orgID := strings.TrimSpace(c.credentials.OrgID)
		if orgID == "" {
			return nil, fmt.Errorf("org ID is required")
		}
		req.Header.Set("X-AP-Context", "orgId="+orgID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseError(respBody, resp.StatusCode)
	}
	if len(strings.TrimSpace(string(respBody))) == 0 {
		return RawResponse(`{"data":null}`), nil
	}
	return RawResponse(respBody), nil
}

func (c *Client) requestURL(path string, query url.Values) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if parsed.IsAbs() {
		base, err := url.Parse(c.baseURL)
		if err != nil {
			return "", err
		}
		if parsed.Scheme != "https" || parsed.Host != base.Host || !strings.HasPrefix(parsed.Path, base.Path+"v5/") {
			return "", fmt.Errorf("--path must be an Apple Ads v5 URL")
		}
		if len(query) > 0 {
			values := parsed.Query()
			for key, items := range query {
				for _, item := range items {
					values.Add(key, item)
				}
			}
			parsed.RawQuery = values.Encode()
		}
		return parsed.String(), nil
	}
	clean := strings.TrimPrefix(path, "/")
	if !strings.HasPrefix(clean, "v5/") {
		return "", fmt.Errorf("--path must start with v5/")
	}
	rel, err := url.Parse(clean)
	if err != nil {
		return "", err
	}
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(rel)
	if len(query) > 0 {
		values := resolved.Query()
		for key, items := range query {
			for _, item := range items {
				values.Add(key, item)
			}
		}
		resolved.RawQuery = values.Encode()
	}
	return resolved.String(), nil
}

func expandPath(path string, params map[string]string) (string, error) {
	result := path
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			return "", fmt.Errorf("%s is required", key)
		}
		result = strings.ReplaceAll(result, "{"+key+"}", url.PathEscape(value))
	}
	if strings.Contains(result, "{") || strings.Contains(result, "}") {
		return "", fmt.Errorf("missing path parameter for %s", path)
	}
	return result, nil
}
