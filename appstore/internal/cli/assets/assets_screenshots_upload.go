package assets

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
)

func normalizeScreenshotDisplayType(input string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(input))
	if value == "" {
		return "", fmt.Errorf("device type is required")
	}
	if !strings.HasPrefix(value, "APP_") && !strings.HasPrefix(value, "IMESSAGE_") {
		value = "APP_" + value
	}
	if !asc.IsValidScreenshotDisplayType(value) {
		return "", fmt.Errorf("unsupported screenshot display type %q", value)
	}
	return value, nil
}

func validateScreenshotDimensions(files []string, displayType string) error {
	for _, filePath := range files {
		if err := asc.ValidateScreenshotDimensions(filePath, displayType); err != nil {
			return err
		}
	}
	return nil
}

func uploadScreenshots(ctx context.Context, client *asc.Client, localizationID, displayType string, files []string, skipExisting, replace, dryRun bool) (asc.AppScreenshotUploadResult, error) {
	result, err := uploadScreenshotsWithConfig(ctx, screenshotUploadConfig[asc.AppScreenshotUploadResult]{
		Client:         client,
		LocalizationID: localizationID,
		DisplayType:    displayType,
		Files:          files,
		SkipExisting:   skipExisting,
		Replace:        replace,
		DryRun:         dryRun,
		RequestContext: shared.ContextWithTimeout,
		UploadContext:  contextWithAssetUploadTimeout,
		Access:         appStoreVersionScreenshotSetAccess,
		BuildResult: func(localizationID string, set asc.Resource[asc.AppScreenshotSetAttributes], dryRun bool, results []asc.AssetUploadResultItem) asc.AppScreenshotUploadResult {
			return buildAppScreenshotUploadResult(localizationID, set, dryRun, results)
		},
	})
	return result, err
}

func findScreenshotSetWithAccess(ctx context.Context, client *asc.Client, localizationID, displayType string, access ScreenshotSetAccess) (asc.Resource[asc.AppScreenshotSetAttributes], error) {
	if access.List == nil {
		return asc.Resource[asc.AppScreenshotSetAttributes]{}, fmt.Errorf("screenshot set list function is required")
	}

	resp, err := access.List(ctx, client, localizationID)
	if err != nil {
		return asc.Resource[asc.AppScreenshotSetAttributes]{}, err
	}
	for _, set := range resp.Data {
		if strings.EqualFold(set.Attributes.ScreenshotDisplayType, displayType) {
			return set, nil
		}
	}
	return asc.Resource[asc.AppScreenshotSetAttributes]{
		Attributes: asc.AppScreenshotSetAttributes{ScreenshotDisplayType: displayType},
	}, nil
}

func ensureScreenshotSetWithAccess(ctx context.Context, client *asc.Client, localizationID, displayType string, access ScreenshotSetAccess) (asc.Resource[asc.AppScreenshotSetAttributes], error) {
	if access.Create == nil {
		return asc.Resource[asc.AppScreenshotSetAttributes]{}, fmt.Errorf("screenshot set create function is required")
	}

	set, err := findScreenshotSetWithAccess(ctx, client, localizationID, displayType, access)
	if err != nil {
		return asc.Resource[asc.AppScreenshotSetAttributes]{}, err
	}
	if set.ID != "" {
		return set, nil
	}

	created, err := access.Create(ctx, client, localizationID, displayType)
	if err != nil {
		return asc.Resource[asc.AppScreenshotSetAttributes]{}, err
	}
	return created.Data, nil
}

func uploadScreenshotsWithConfig[T any](ctx context.Context, cfg screenshotUploadConfig[T]) (T, error) {
	var zero T

	if cfg.Client == nil {
		return zero, fmt.Errorf("client is required")
	}
	if cfg.BuildResult == nil {
		return zero, fmt.Errorf("build result function is required")
	}
	if cfg.RequestContext == nil {
		cfg.RequestContext = shared.ContextWithTimeout
	}
	if cfg.UploadContext == nil {
		cfg.UploadContext = contextWithAssetUploadTimeout
	}

	sourceRootPath := ""
	if len(cfg.Files) > 0 {
		var err error
		sourceRootPath, err = resolveScreenshotUploadRoot(cfg.RootPath, cfg.Files)
		if err != nil {
			return zero, fmt.Errorf("resolve screenshot source root: %w", err)
		}
	}

	requestCtx, reqCancel := cfg.RequestContext(ctx)
	var (
		set asc.Resource[asc.AppScreenshotSetAttributes]
		err error
	)
	if cfg.DryRun {
		set, err = findScreenshotSetWithAccess(requestCtx, cfg.Client, cfg.LocalizationID, cfg.DisplayType, cfg.Access)
	} else {
		set, err = ensureScreenshotSetWithAccess(requestCtx, cfg.Client, cfg.LocalizationID, cfg.DisplayType, cfg.Access)
	}
	reqCancel()
	if err != nil {
		return zero, err
	}

	existingScreenshots := make([]asc.Resource[asc.AppScreenshotAttributes], 0)
	if (cfg.SkipExisting || cfg.Replace || (!cfg.Replace && len(cfg.Files) > 0)) && set.ID != "" {
		fetchCtx, fetchCancel := cfg.RequestContext(ctx)
		existingResp, err := cfg.Client.GetAppScreenshots(fetchCtx, set.ID)
		fetchCancel()
		if err != nil {
			return zero, err
		}
		existingScreenshots = existingResp.Data
	}

	skippedResults := make([]asc.AssetUploadResultItem, 0)
	files := cfg.Files
	if cfg.SkipExisting {
		var filterErr error
		files, skippedResults, filterErr = filterExistingScreenshotFiles(cfg.Files, existingScreenshots)
		if filterErr != nil {
			return zero, filterErr
		}
	}
	files, err = limitScreenshotUploadFilesForExistingSet(files, cfg.MaxScreenshots, existingScreenshots, cfg.Replace, set.ID)
	if err != nil {
		return zero, err
	}

	if cfg.DryRun {
		results := make([]asc.AssetUploadResultItem, 0, len(skippedResults)+len(files)+len(existingScreenshots))
		if cfg.Replace {
			for _, screenshot := range existingScreenshots {
				results = append(results, asc.AssetUploadResultItem{
					FileName: screenshot.Attributes.FileName,
					AssetID:  screenshot.ID,
					State:    "would-delete",
				})
			}
		}
		for _, filePath := range files {
			results = append(results, asc.AssetUploadResultItem{
				FileName: filepath.Base(filePath),
				FilePath: filePath,
				State:    "would-upload",
			})
		}
		results = append(results, skippedResults...)
		return cfg.BuildResult(cfg.LocalizationID, set, true, results), nil
	}

	uploadCtx, cancel := cfg.UploadContext(ctx)
	defer cancel()

	if cfg.Replace {
		if err := deleteExistingScreenshots(uploadCtx, cfg.Client, existingScreenshots); err != nil {
			return zero, err
		}
	}

	results := make([]asc.AssetUploadResultItem, 0, len(skippedResults)+len(files))
	if len(files) > 0 {
		uploadedResults, err := uploadScreenshotsToSetFromRoot(uploadCtx, cfg.Client, set.ID, files, sourceRootPath, !cfg.Replace)
		if err != nil {
			return zero, err
		}
		results = append(results, uploadedResults...)
	}
	if cfg.SkipExisting && len(skippedResults) > 0 {
		if _, err := syncSkippedScreenshotOrder(uploadCtx, cfg.Client, set.ID, cfg.Files, skippedResults, results); err != nil {
			results = append(skippedResults, results...)
			return cfg.BuildResult(cfg.LocalizationID, set, false, results), err
		}
	}
	results = append(skippedResults, results...)

	return cfg.BuildResult(cfg.LocalizationID, set, false, results), nil
}

func deleteExistingScreenshots(ctx context.Context, client *asc.Client, screenshots []asc.Resource[asc.AppScreenshotAttributes]) error {
	for _, screenshot := range screenshots {
		if err := client.DeleteAppScreenshot(ctx, screenshot.ID); err != nil {
			return err
		}
	}
	return nil
}

func filterExistingScreenshotFiles(files []string, screenshots []asc.Resource[asc.AppScreenshotAttributes]) ([]string, []asc.AssetUploadResultItem, error) {
	existingByChecksum := make(map[string]asc.Resource[asc.AppScreenshotAttributes], len(screenshots))
	for _, screenshot := range screenshots {
		checksum := strings.TrimSpace(screenshot.Attributes.SourceFileChecksum)
		if checksum == "" {
			continue
		}
		if _, exists := existingByChecksum[checksum]; !exists {
			existingByChecksum[checksum] = screenshot
		}
	}

	filtered := make([]string, 0, len(files))
	skipped := make([]asc.AssetUploadResultItem, 0)
	for _, filePath := range files {
		checksum, err := screenshotFileChecksumFunc(filePath)
		if err != nil {
			return nil, nil, err
		}
		if existing, exists := existingByChecksum[checksum]; exists {
			skipped = append(skipped, asc.AssetUploadResultItem{
				FileName: filepath.Base(filePath),
				FilePath: filePath,
				AssetID:  existing.ID,
				State:    "skipped",
				Skipped:  true,
			})
			continue
		}
		filtered = append(filtered, filePath)
	}

	return filtered, skipped, nil
}

func syncSkippedScreenshotOrder(ctx context.Context, client *asc.Client, setID string, files []string, skippedResults, uploadedResults []asc.AssetUploadResultItem) ([]string, error) {
	currentOrder, err := GetOrderedAppScreenshotIDs(ctx, client, setID)
	if err != nil {
		return nil, err
	}

	orderedIDs := orderScreenshotIDsForLocalFiles(currentOrder, files, skippedResults, uploadedResults)
	if sameScreenshotIDOrder(currentOrder, orderedIDs) {
		return orderedIDs, nil
	}
	return orderedIDs, SetOrderedAppScreenshots(ctx, client, setID, orderedIDs)
}

func orderScreenshotIDsForLocalFiles(currentOrder []string, files []string, skippedResults, uploadedResults []asc.AssetUploadResultItem) []string {
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

func sameScreenshotIDOrder(a, b []string) bool {
	a = normalizeScreenshotIDs(a)
	b = normalizeScreenshotIDs(b)
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

func computeFileChecksum(filePath string) (string, error) {
	file, err := shared.OpenExistingNoFollow(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	checksum, err := asc.ComputeChecksumFromReader(file, asc.ChecksumAlgorithmMD5)
	if err != nil {
		return "", err
	}
	return checksum.Hash, nil
}

func computeFileChecksumInRoot(rootPath, filePath string) (string, error) {
	root, err := rootfs.New(rootPath)
	if err != nil {
		return "", err
	}
	file, err := root.OpenFile(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	checksum, err := asc.ComputeChecksumFromReader(file, asc.ChecksumAlgorithmMD5)
	if err != nil {
		return "", err
	}
	return checksum.Hash, nil
}

func uploadScreenshotAsset(ctx context.Context, client *asc.Client, setID, sourceRootPath, filePath string) (asc.AssetUploadResultItem, screenshotPendingAsset, error) {
	root, err := rootfs.New(sourceRootPath)
	if err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}
	sourcePath, err := filepath.Abs(filePath)
	if err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}
	file, err := root.OpenFile(sourcePath)
	if err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}
	defer file.Close()
	return uploadScreenshotAssetFromFile(ctx, client, setID, filePath, file)
}

func uploadScreenshotAssetFromFile(ctx context.Context, client *asc.Client, setID, filePath string, file *os.File) (asc.AssetUploadResultItem, screenshotPendingAsset, error) {
	if file == nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, fmt.Errorf("screenshot file is required")
	}
	info, err := file.Stat()
	if err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}
	if !info.Mode().IsRegular() {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, fmt.Errorf("expected regular file: %q", filePath)
	}
	if info.Size() <= 0 {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, fmt.Errorf("file is empty: %q", filePath)
	}
	const maxScreenshotUploadFileSize = int64(1024 * 1024 * 1024)
	if info.Size() > maxScreenshotUploadFileSize {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, fmt.Errorf("file size exceeds %d bytes: %q", maxScreenshotUploadFileSize, filePath)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}

	checksum, err := asc.ComputeChecksumFromReader(file, asc.ChecksumAlgorithmMD5)
	if err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}

	created, err := client.CreateAppScreenshot(ctx, setID, info.Name(), info.Size())
	if err != nil {
		return asc.AssetUploadResultItem{}, screenshotPendingAsset{}, err
	}
	pending := screenshotPendingAsset{
		FileName: info.Name(),
		FilePath: filePath,
		AssetID:  created.Data.ID,
		Checksum: checksum.Hash,
		State:    "AWAITING_UPLOAD",
	}
	if len(created.Data.Attributes.UploadOperations) == 0 {
		return asc.AssetUploadResultItem{}, pending, fmt.Errorf("no upload operations returned for %q", info.Name())
	}

	if err := asc.UploadAssetFromFile(ctx, file, info.Size(), created.Data.Attributes.UploadOperations); err != nil {
		return asc.AssetUploadResultItem{}, pending, err
	}
	pending.State = "UPLOADED"

	updated, err := client.UpdateAppScreenshot(ctx, created.Data.ID, true, checksum.Hash)
	if err != nil {
		return asc.AssetUploadResultItem{}, pending, err
	}
	pending.State = "UPLOAD_COMPLETE"
	if updated.Data.Attributes.AssetDeliveryState != nil && strings.TrimSpace(updated.Data.Attributes.AssetDeliveryState.State) != "" {
		pending.State = strings.ToUpper(strings.TrimSpace(updated.Data.Attributes.AssetDeliveryState.State))
	}

	state, err := waitForScreenshotDelivery(ctx, client, created.Data.ID)
	if strings.TrimSpace(state) != "" {
		pending.State = strings.ToUpper(strings.TrimSpace(state))
	}
	if err != nil {
		return asc.AssetUploadResultItem{}, pending, err
	}

	return asc.AssetUploadResultItem{
		FileName: info.Name(),
		FilePath: filePath,
		AssetID:  created.Data.ID,
		State:    state,
	}, screenshotPendingAsset{}, nil
}

// UploadScreenshotAsset uploads a screenshot file to a set.
func UploadScreenshotAsset(ctx context.Context, client *asc.Client, setID, filePath string) (asc.AssetUploadResultItem, error) {
	sourceRootPath, err := resolveScreenshotUploadRoot("", []string{filePath})
	if err != nil {
		return asc.AssetUploadResultItem{}, err
	}
	result, _, err := uploadScreenshotAsset(ctx, client, setID, sourceRootPath, filePath)
	return result, err
}

// UploadScreenshotAssetFromFile uploads from an already-open, validated source
// handle. Callers that discover files under a rooted filesystem can retain the
// handle so a later pathname replacement cannot redirect the upload.
func UploadScreenshotAssetFromFile(ctx context.Context, client *asc.Client, setID, filePath string, file *os.File) (asc.AssetUploadResultItem, error) {
	result, _, err := uploadScreenshotAssetFromFile(ctx, client, setID, filePath, file)
	return result, err
}

func waitForScreenshotDelivery(ctx context.Context, client *asc.Client, screenshotID string) (string, error) {
	return waitForAssetDeliveryState(ctx, screenshotID, func(ctx context.Context) (*asc.AssetDeliveryState, error) {
		resp, err := client.GetAppScreenshot(ctx, screenshotID)
		if err != nil {
			return nil, err
		}
		return resp.Data.Attributes.AssetDeliveryState, nil
	})
}
