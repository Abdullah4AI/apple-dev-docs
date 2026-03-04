package shared

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"

// BuildAppKeywordsResponse converts a keyword list into the standard
// appKeywords response shape used by multiple keyword mutation commands.
func BuildAppKeywordsResponse(keywords []string) *asc.AppKeywordsResponse {
	resp := &asc.AppKeywordsResponse{
		Data: make([]asc.Resource[asc.AppKeywordAttributes], 0, len(keywords)),
	}
	for _, keyword := range keywords {
		resp.Data = append(resp.Data, asc.Resource[asc.AppKeywordAttributes]{
			Type:       asc.ResourceTypeAppKeywords,
			ID:         keyword,
			Attributes: asc.AppKeywordAttributes{},
		})
	}
	return resp
}
