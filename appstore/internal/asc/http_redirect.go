package asc

import "net/http"

func clientWithoutRedirects(client *http.Client) *http.Client {
	if client == nil {
		client = newDefaultHTTPClient(ResolveTimeout())
	}
	safeClient := *client
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		// A redirect can disclose a signed URL through Referer and 307/308 can
		// replay an upload body to an attacker-controlled destination.
		return http.ErrUseLastResponse
	}
	return &safeClient
}
