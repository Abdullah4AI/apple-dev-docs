package itunes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PublicSearchPlatform identifies a public App Store software surface.
type PublicSearchPlatform string

const (
	// PublicSearchPlatformIOS selects the iOS App Store software surface.
	PublicSearchPlatformIOS PublicSearchPlatform = "IOS"
	// PublicSearchPlatformTVOS selects the Apple TV App Store software surface.
	PublicSearchPlatformTVOS PublicSearchPlatform = "TV_OS"

	appleTVStorefrontSoftwareKind = "33"
	appleTVSoftwareEntity         = "tvSoftware"
	storefrontSearchPath          = "/WebObjects/MZStore.woa/wa/search"
)

// PublicRankResult reports whether an app appears in an ordered result window.
type PublicRankResult struct {
	Found       bool
	Rank        *int
	ResultCount int
}

// RankApp reports an app's position in a public App Store search result window.
func (c *Client) RankApp(
	ctx context.Context,
	appID string,
	term string,
	country string,
	platform PublicSearchPlatform,
) (PublicRankResult, error) {
	switch platform {
	case PublicSearchPlatformIOS:
		results, err := c.SearchApps(ctx, term, country, 200)
		if err != nil {
			return PublicRankResult{}, fmt.Errorf("rank iOS apps: %w", err)
		}

		orderedIDs := make([]string, 0, len(results))
		for _, result := range results {
			orderedIDs = append(orderedIDs, strconv.FormatInt(result.AppID, 10))
		}
		return rankOrderedAppIDs(appID, orderedIDs), nil
	case PublicSearchPlatformTVOS:
		return c.rankTVApp(ctx, appID, term, country)
	default:
		return PublicRankResult{}, fmt.Errorf("unsupported public search platform %q", platform)
	}
}

type storefrontSearchResponse struct {
	PageData struct {
		Bubbles []struct {
			Results []struct {
				ID     string `json:"id"`
				Entity string `json:"entity"`
			} `json:"results"`
		} `json:"bubbles"`
	} `json:"pageData"`
}

func (c *Client) rankTVApp(ctx context.Context, appID, term, country string) (PublicRankResult, error) {
	normalizedCountry, err := NormalizeCountryCode(country)
	if err != nil {
		return PublicRankResult{}, err
	}
	storefrontID := strings.TrimSpace(Storefronts[normalizedCountry])
	if storefrontID == "" {
		return PublicRankResult{}, fmt.Errorf("TV_OS ranking is unavailable for storefront %q", strings.ToUpper(normalizedCountry))
	}

	base, err := url.Parse(c.storefrontSearchBaseURL())
	if err != nil {
		return PublicRankResult{}, fmt.Errorf("invalid storefront search base URL: %w", err)
	}
	query := url.Values{}
	query.Set("clientApplication", "Software")
	query.Set("src", "hint")
	query.Set("submit", "edit")
	query.Set("term", strings.TrimSpace(term))

	reqURL := *base
	reqURL.Path = strings.TrimRight(reqURL.Path, "/") + storefrontSearchPath
	reqURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return PublicRankResult{}, fmt.Errorf("failed to create storefront search request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Apple-Store-Front", storefrontID+","+appleTVStorefrontSoftwareKind)

	var payload storefrontSearchResponse
	if err := c.do(ctx, "storefront search", req, func(resp *http.Response) error {
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return fmt.Errorf("failed to parse storefront search response: %w", err)
		}
		return nil
	}); err != nil {
		return PublicRankResult{}, err
	}
	if payload.PageData.Bubbles == nil {
		return PublicRankResult{}, fmt.Errorf("invalid storefront search response: missing pageData.bubbles")
	}

	orderedIDs := make([]string, 0)
	seenIDs := make(map[string]struct{})
	totalResults := 0
	tvResults := 0
	for bubbleIndex, bubble := range payload.PageData.Bubbles {
		if bubble.Results == nil {
			return PublicRankResult{}, fmt.Errorf("invalid storefront search response: missing pageData.bubbles[%d].results", bubbleIndex)
		}
		for _, result := range bubble.Results {
			totalResults++
			if result.Entity != appleTVSoftwareEntity {
				continue
			}
			tvResults++
			resultID := strings.TrimSpace(result.ID)
			if resultID == "" {
				return PublicRankResult{}, fmt.Errorf("invalid storefront search response: tvSoftware result is missing an ID")
			}
			canonicalID := canonicalizeLookupID(resultID)
			if _, ok := seenIDs[canonicalID]; ok {
				continue
			}
			seenIDs[canonicalID] = struct{}{}
			orderedIDs = append(orderedIDs, canonicalID)
		}
	}
	if totalResults > 0 && tvResults == 0 {
		return PublicRankResult{}, fmt.Errorf("invalid storefront search response: expected tvSoftware results")
	}

	return rankOrderedAppIDs(appID, orderedIDs), nil
}

func rankOrderedAppIDs(appID string, orderedIDs []string) PublicRankResult {
	result := PublicRankResult{ResultCount: len(orderedIDs)}
	targetID := canonicalizeLookupID(appID)
	for index, candidateID := range orderedIDs {
		if canonicalizeLookupID(candidateID) != targetID {
			continue
		}
		rank := index + 1
		result.Found = true
		result.Rank = &rank
		return result
	}
	return result
}
