package asc

import "strings"

// AppWallEntry represents one row in apps wall output.
type AppWallEntry struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Platform    []string `json:"platform,omitempty"`
	Creator     string   `json:"creator,omitempty"`
	AppStoreURL string   `json:"appStoreUrl"`
	ReleaseDate string   `json:"releaseDate,omitempty"`
}

// AppsWallResult is the response payload for apps wall output.
type AppsWallResult struct {
	Data []AppWallEntry `json:"data"`
}

func appsWallRows(resp *AppsWallResult) ([]string, [][]string) {
	headers := []string{"App", "Link", "Creator", "Platform"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			compactWhitespace(item.Name),
			item.AppStoreURL,
			compactWhitespace(item.Creator),
			strings.Join(formatWallPlatformsForDisplay(item.Platform), ", "),
		})
	}
	return headers, rows
}

func formatWallPlatformsForDisplay(platforms []string) []string {
	if len(platforms) == 0 {
		return nil
	}
	display := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		switch platform {
		case "IOS":
			display = append(display, "iOS")
		case "MAC_OS":
			display = append(display, "macOS")
		case "TV_OS":
			display = append(display, "tvOS")
		case "VISION_OS":
			display = append(display, "visionOS")
		default:
			display = append(display, platform)
		}
	}
	return display
}
