package assets

import (
	"context"
	"fmt"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// collectOrderedLinkageIDs walks every linkage page and returns the linked
// resource IDs in the order App Store Connect reports them.
func collectOrderedLinkageIDs(ctx context.Context, firstPage *asc.LinkagesResponse, next func(context.Context, string) (asc.PaginatedResponse, error)) ([]string, error) {
	if firstPage == nil {
		return nil, fmt.Errorf("linkage response is required")
	}

	orderedIDs := make([]string, 0, len(firstPage.Data))
	err := asc.PaginateEach(ctx, firstPage, next, func(page asc.PaginatedResponse) error {
		linkages, ok := page.(*asc.LinkagesResponse)
		if !ok {
			return fmt.Errorf("unexpected relationship response type %T", page)
		}
		for _, item := range linkages.Data {
			orderedIDs = appendUniqueAssetID(orderedIDs, item.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return orderedIDs, nil
}

// orderAssetIDsForLocalFiles orders asset IDs by the local file order of the
// current run and appends any remaining remote IDs in their existing order.
func orderAssetIDsForLocalFiles(currentOrder []string, files []string, skippedResults, uploadedResults []asc.AssetUploadResultItem) []string {
	skippedByPath := make(map[string]string, len(skippedResults))
	for _, item := range skippedResults {
		if strings.TrimSpace(item.AssetID) == "" {
			continue
		}
		skippedByPath[item.FilePath] = item.AssetID
	}
	uploadedByPath := make(map[string]string, len(uploadedResults))
	for _, item := range uploadedResults {
		if strings.TrimSpace(item.AssetID) == "" {
			continue
		}
		uploadedByPath[item.FilePath] = item.AssetID
	}

	orderedIDs := make([]string, 0, len(currentOrder)+len(uploadedResults))
	seen := make(map[string]struct{}, len(currentOrder)+len(uploadedResults))
	for _, filePath := range files {
		id := skippedByPath[filePath]
		if id == "" {
			id = uploadedByPath[filePath]
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		orderedIDs = append(orderedIDs, id)
	}
	for _, id := range currentOrder {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		orderedIDs = append(orderedIDs, id)
	}

	return orderedIDs
}

// appendUploadedAssetIDs keeps the remote order of assets that already existed
// before this run and appends the newly uploaded assets in upload order.
func appendUploadedAssetIDs(currentOrder []string, uploadedResults []asc.AssetUploadResultItem) []string {
	uploadedIDs := make(map[string]struct{}, len(uploadedResults))
	for _, item := range uploadedResults {
		if id := strings.TrimSpace(item.AssetID); id != "" {
			uploadedIDs[id] = struct{}{}
		}
	}

	orderedIDs := make([]string, 0, len(currentOrder)+len(uploadedResults))
	for _, id := range currentOrder {
		if _, uploaded := uploadedIDs[strings.TrimSpace(id)]; uploaded {
			continue
		}
		orderedIDs = appendUniqueAssetID(orderedIDs, id)
	}
	for _, item := range uploadedResults {
		orderedIDs = appendUniqueAssetID(orderedIDs, item.AssetID)
	}

	return orderedIDs
}

func sameAssetIDOrder(a, b []string) bool {
	a = normalizeAssetIDs(a)
	b = normalizeAssetIDs(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func normalizeAssetIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}

func appendUniqueAssetID(ids []string, id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
