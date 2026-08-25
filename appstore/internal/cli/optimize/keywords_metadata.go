package optimize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/itunes"
)

// keywordLookupChunkSize bounds one public lookup request. Apple's lookup
// endpoint accepts a batch of IDs, and competitor metadata is deduplicated
// across every keyword before it is requested.
const keywordLookupChunkSize = 50

const keywordLookupDefaultBaseURL = "https://itunes.apple.com"

// publicAppMetadata is the subset of public lookup metadata the difficulty
// engine needs beyond what the search endpoint already returns.
//
// internal/itunes.App intentionally does not expose release dates, and this
// package must not widen that shared type while other work is in flight, so
// the two release dates are read here from the same public endpoint using the
// shared client's base URL and HTTP client.
type publicAppMetadata struct {
	AppID                     string
	Name                      string
	PublisherName             string
	AverageUserRating         float64
	UserRatingCount           int64
	ReleaseDate               string
	CurrentVersionReleaseDate string
}

type publicLookupResponse struct {
	Results []struct {
		TrackID                   int64   `json:"trackId"`
		TrackName                 string  `json:"trackName"`
		SellerName                string  `json:"sellerName"`
		AverageUserRating         float64 `json:"averageUserRating"`
		UserRatingCount           int64   `json:"userRatingCount"`
		ReleaseDate               string  `json:"releaseDate"`
		CurrentVersionReleaseDate string  `json:"currentVersionReleaseDate"`
	} `json:"results"`
}

// fetchPublicAppMetadata reads release dates for a batch of app IDs from
// Apple's public lookup endpoint. It returns requested rows keyed by app ID;
// invalid date fields are blanked while individually valid dates are retained
// so callers can preserve partial scoring inputs and report incomplete coverage.
func fetchPublicAppMetadata(
	ctx context.Context,
	client *itunes.Client,
	ids []string,
	country string,
) (map[string]publicAppMetadata, error) {
	if len(ids) == 0 {
		return map[string]publicAppMetadata{}, nil
	}
	if len(ids) > keywordLookupChunkSize {
		return nil, fmt.Errorf("lookup accepts at most %d app IDs per request", keywordLookupChunkSize)
	}

	query := url.Values{}
	query.Set("id", strings.Join(ids, ","))
	query.Set("entity", "software")
	if trimmedCountry := strings.TrimSpace(country); trimmedCountry != "" {
		query.Set("country", trimmedCountry)
	}

	base := keywordLookupDefaultBaseURL
	if client != nil && strings.TrimSpace(client.BaseURL) != "" {
		base = strings.TrimRight(strings.TrimSpace(client.BaseURL), "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/lookup?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	httpClient := http.DefaultClient
	if client != nil && client.HTTPClient != nil {
		httpClient = client.HTTPClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lookup request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lookup request returned status %d", resp.StatusCode)
	}

	var payload publicLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to parse lookup response: %w", err)
	}

	requested := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			requested[trimmed] = struct{}{}
		}
	}
	metadata := make(map[string]publicAppMetadata, len(payload.Results))
	for _, result := range payload.Results {
		if result.TrackID == 0 {
			continue
		}
		appID := strconv.FormatInt(result.TrackID, 10)
		if _, ok := requested[appID]; !ok {
			continue
		}
		releaseDate := strings.TrimSpace(result.ReleaseDate)
		currentVersionReleaseDate := strings.TrimSpace(result.CurrentVersionReleaseDate)
		metadata[appID] = publicAppMetadata{
			AppID:                     appID,
			Name:                      strings.TrimSpace(result.TrackName),
			PublisherName:             strings.TrimSpace(result.SellerName),
			AverageUserRating:         result.AverageUserRating,
			UserRatingCount:           result.UserRatingCount,
			ReleaseDate:               validPublicDate(releaseDate),
			CurrentVersionReleaseDate: validPublicDate(currentVersionReleaseDate),
		}
	}
	return metadata, nil
}

func validPublicDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if _, ok := parsePublicDate(trimmed); !ok {
		return ""
	}
	return trimmed
}

// chunkAppIDs splits deduplicated app IDs into lookup-sized batches.
func chunkAppIDs(ids []string, size int) [][]string {
	if size < 1 {
		size = 1
	}
	chunks := make([][]string, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := min(start+size, len(ids))
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}
