package metadata

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

const (
	metadataURLCheckConcurrency = 4
	metadataURLCheckTimeout     = 10 * time.Second
	metadataURLMaxRedirects     = 10
)

var errUnsafeMetadataURLTarget = errors.New("metadata URL target is not a public internet address")

var newMetadataURLChecker = func() metadataURLChecker {
	return newHTTPMetadataURLChecker()
}

type metadataURLCheckResult struct {
	FinalURL   *url.URL
	StatusCode int
}

type metadataURLChecker interface {
	Check(context.Context, string) (metadataURLCheckResult, error)
}

type metadataURLTarget struct {
	rawURL  string
	scope   string
	file    string
	locale  string
	version string
	field   string
	label   string
}

type metadataURLCheckOutcome struct {
	result metadataURLCheckResult
	err    error
}

func metadataURLTargets(scope, file, locale, version string, fields []metadataURLField) []metadataURLTarget {
	targets := make([]metadataURLTarget, 0, len(fields))
	for _, field := range fields {
		rawURL := strings.TrimSpace(field.value)
		if rawURL == "" || !validation.IsValidHTTPURL(rawURL) {
			continue
		}
		targets = append(targets, metadataURLTarget{
			rawURL:  rawURL,
			scope:   scope,
			file:    file,
			locale:  locale,
			version: version,
			field:   field.field,
			label:   field.label,
		})
	}
	return targets
}

func metadataURLCheckIssues(ctx context.Context, checker metadataURLChecker, targets []metadataURLTarget) ([]ValidateIssue, error) {
	uniqueURLs := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		uniqueURLs[target.rawURL] = struct{}{}
	}
	urls := make([]string, 0, len(uniqueURLs))
	for rawURL := range uniqueURLs {
		urls = append(urls, rawURL)
	}
	sort.Strings(urls)

	outcomes := make(map[string]metadataURLCheckOutcome, len(urls))
	var outcomesMu sync.Mutex
	jobs := make(chan string)
	var workers sync.WaitGroup
	workerCount := min(metadataURLCheckConcurrency, len(urls))
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for rawURL := range jobs {
				result, err := checker.Check(ctx, rawURL)
				outcomesMu.Lock()
				outcomes[rawURL] = metadataURLCheckOutcome{result: result, err: err}
				outcomesMu.Unlock()
			}
		}()
	}
	for _, rawURL := range urls {
		jobs <- rawURL
	}
	close(jobs)
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	issues := make([]ValidateIssue, 0, len(targets))
	for _, target := range targets {
		outcome := outcomes[target.rawURL]
		for _, message := range metadataURLCheckMessages(target, outcome) {
			issues = append(issues, ValidateIssue{
				Scope:    target.scope,
				File:     target.file,
				Locale:   target.locale,
				Version:  target.version,
				Field:    target.field,
				Severity: issueSeverityWarning,
				Message:  message,
			})
		}
	}
	return issues, nil
}

func metadataURLCheckMessages(target metadataURLTarget, outcome metadataURLCheckOutcome) []string {
	if outcome.err != nil {
		reason := "request failed"
		if errors.Is(outcome.err, errUnsafeMetadataURLTarget) {
			reason = "target is not a public internet address"
		} else if errors.Is(outcome.err, context.DeadlineExceeded) || isTimeoutError(outcome.err) {
			reason = "request timed out"
		}
		return []string{fmt.Sprintf("%s could not be checked: %s", target.label, reason)}
	}
	if outcome.result.FinalURL == nil {
		return []string{fmt.Sprintf("%s could not be checked: request failed", target.label)}
	}

	initialURL, err := url.Parse(target.rawURL)
	if err != nil {
		return nil
	}
	messages := make([]string, 0, 2)
	initialHost := strings.ToLower(initialURL.Hostname())
	finalHost := strings.ToLower(outcome.result.FinalURL.Hostname())
	if initialHost != finalHost {
		messages = append(messages, fmt.Sprintf("%s redirects to a different host (%s -> %s)", target.label, initialHost, finalHost))
	}
	if outcome.result.StatusCode < http.StatusOK || outcome.result.StatusCode >= http.StatusMultipleChoices {
		return append(messages, fmt.Sprintf("%s returned HTTP %d", target.label, outcome.result.StatusCode))
	}
	finalPath := outcome.result.FinalURL.EscapedPath()
	if (finalPath == "" || finalPath == "/") && outcome.result.FinalURL.RawQuery == "" && outcome.result.FinalURL.Fragment == "" {
		messages = append(messages, fmt.Sprintf("%s resolves to a site root instead of a dedicated page", target.label))
	}
	return messages
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

type httpMetadataURLChecker struct {
	client *http.Client
}

func newHTTPMetadataURLChecker() *httpMetadataURLChecker {
	timeout := asc.ResolveTimeoutWithDefault(metadataURLCheckTimeout)
	dialer := &net.Dialer{
		Timeout:        timeout,
		KeepAlive:      30 * time.Second,
		ControlContext: publicMetadataURLDialControl,
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = dialer.DialContext
	transport.DisableKeepAlives = true
	transport.MaxConnsPerHost = metadataURLCheckConcurrency
	transport.ResponseHeaderTimeout = timeout
	transport.TLSHandshakeTimeout = timeout
	transport.MaxResponseHeaderBytes = 1 << 20

	return &httpMetadataURLChecker{client: &http.Client{
		Transport:     transport,
		Timeout:       timeout,
		CheckRedirect: metadataURLRedirectPolicy,
	}}
}

func (c *httpMetadataURLChecker) Check(ctx context.Context, rawURL string) (metadataURLCheckResult, error) {
	requestCtx, cancel := shared.ContextWithTimeout(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return metadataURLCheckResult{}, err
	}
	req.Header.Set("User-Agent", "asc metadata validate")

	resp, err := c.client.Do(req)
	if err != nil {
		return metadataURLCheckResult{}, err
	}
	defer resp.Body.Close()
	return metadataURLCheckResult{FinalURL: resp.Request.URL, StatusCode: resp.StatusCode}, nil
}

func metadataURLRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= metadataURLMaxRedirects {
		return fmt.Errorf("metadata URL exceeded %d redirects", metadataURLMaxRedirects)
	}
	if !isHTTPMetadataURL(req.URL) {
		return errUnsafeMetadataURLTarget
	}
	if ip, err := netip.ParseAddr(req.URL.Hostname()); err == nil && !isPublicMetadataIP(ip) {
		return errUnsafeMetadataURLTarget
	}
	return nil
}

func isHTTPMetadataURL(parsed *url.URL) bool {
	if parsed == nil || parsed.Hostname() == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func publicMetadataURLDialControl(_ context.Context, _, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return errUnsafeMetadataURLTarget
	}
	parsed, err := netip.ParseAddr(host)
	if err != nil || !isPublicMetadataIP(parsed) {
		return errUnsafeMetadataURLTarget
	}
	return nil
}

// These IANA special-purpose ranges can still satisfy netip.Addr.IsGlobalUnicast.
// Translation prefixes stay blocked because they can encode non-public IPv4 targets.
var reservedMetadataIPPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::ffff:0:0:0/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3ffe::/16"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/8"),
	netip.MustParsePrefix("fec0::/10"),
}

// Check these IANA globally reachable allocations before their broader parent ranges.
var globallyReachableMetadataIPExceptions = []netip.Prefix{
	netip.MustParsePrefix("192.0.0.9/32"),
	netip.MustParsePrefix("192.0.0.10/32"),
	netip.MustParsePrefix("2001:1::1/128"),
	netip.MustParsePrefix("2001:1::2/128"),
	netip.MustParsePrefix("2001:1::3/128"),
	netip.MustParsePrefix("2001:3::/32"),
	netip.MustParsePrefix("2001:4:112::/48"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:30::/28"),
}

func isPublicMetadataIP(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range globallyReachableMetadataIPExceptions {
		if prefix.Contains(address) {
			return true
		}
	}
	for _, prefix := range reservedMetadataIPPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
