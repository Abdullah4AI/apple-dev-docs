package distribute

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	core "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/distribution"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

type publicationStore struct {
	delegate core.ObjectStore
}

func (store publicationStore) Ensure(ctx context.Context, object core.PutObject) (core.StoredObject, error) {
	requestCtx, cancel := shared.ContextWithUploadTimeout(ctx)
	defer cancel()
	return store.delegate.Ensure(requestCtx, object)
}

func (store publicationStore) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	requestCtx, cancel := shared.ContextWithUploadTimeout(ctx)
	defer cancel()
	return store.delegate.PresignGet(requestCtx, key, ttl)
}

type publishRequest struct {
	BundleDir              string
	Endpoint               string
	DownloadEndpoint       string
	Region                 string
	Bucket                 string
	Prefix                 string
	AddressingStyle        string
	Access                 string
	PublicBaseURL          string
	URLTTL                 time.Duration
	DownloadGrace          time.Duration
	VerifyTimeout          time.Duration
	ReceiptPath            string
	LinkPath               string
	PrivateOptionsExplicit bool
	DiagnosticWriter       io.Writer
	ValidateOutput         func() error
}

// publishExecutionResult is safe to return from an in-process workflow. Exact
// bearer links remain exclusively in the protected link artifact.
type publishExecutionResult struct {
	Receipt   core.PublishReceipt `json:"receipt"`
	Recovered bool                `json:"recovered"`
}

// privatePublishRequest intentionally has no access or public-base fields. An
// orchestrator using this adapter cannot opt into unbounded public delivery.
type privatePublishRequest struct {
	BundleDir        string
	Endpoint         string
	DownloadEndpoint string
	Region           string
	Bucket           string
	Prefix           string
	AddressingStyle  string
	URLTTL           time.Duration
	DownloadGrace    time.Duration
	VerifyTimeout    time.Duration
	ReceiptPath      string
	LinkPath         string
	DiagnosticWriter io.Writer
}

func executePrivatePublish(ctx context.Context, request privatePublishRequest) (publishExecutionResult, error) {
	return executePublish(ctx, publishRequest{
		BundleDir:        request.BundleDir,
		Endpoint:         request.Endpoint,
		DownloadEndpoint: request.DownloadEndpoint,
		Region:           request.Region,
		Bucket:           request.Bucket,
		Prefix:           request.Prefix,
		AddressingStyle:  request.AddressingStyle,
		Access:           string(core.AccessPrivate),
		URLTTL:           request.URLTTL,
		DownloadGrace:    request.DownloadGrace,
		VerifyTimeout:    request.VerifyTimeout,
		ReceiptPath:      request.ReceiptPath,
		LinkPath:         request.LinkPath,
		DiagnosticWriter: request.DiagnosticWriter,
	})
}

func executePublish(ctx context.Context, request publishRequest) (publishExecutionResult, error) {
	diagnostic := request.DiagnosticWriter
	if diagnostic == nil {
		diagnostic = io.Discard
	}
	required := []struct{ name, value string }{
		{"bundle-dir", request.BundleDir},
		{"endpoint", request.Endpoint},
		{"region", request.Region},
		{"bucket", request.Bucket},
		{"prefix", request.Prefix},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			return publishExecutionResult{}, shared.MissingRequiredUsageError()
		}
	}
	validatedEndpoint, err := core.ValidateEndpoint(request.Endpoint)
	if err != nil {
		return publishExecutionResult{}, shared.UsageErrorf("--endpoint: %v", err)
	}
	if strings.TrimSpace(request.DownloadEndpoint) != "" {
		if _, err := core.ValidateEndpoint(request.DownloadEndpoint); err != nil {
			return publishExecutionResult{}, shared.UsageErrorf("--download-endpoint: %v", err)
		}
	}
	if request.AddressingStyle != "path" && request.AddressingStyle != "virtual" {
		return publishExecutionResult{}, shared.UsageError("--addressing-style must be path or virtual")
	}
	accessMode := core.Access(strings.ToLower(strings.TrimSpace(request.Access)))
	if accessMode != core.AccessPrivate && accessMode != core.AccessPublic {
		return publishExecutionResult{}, shared.UsageError("--access must be private or public")
	}
	if accessMode == core.AccessPublic {
		if strings.TrimSpace(request.PublicBaseURL) == "" {
			return publishExecutionResult{}, shared.UsageError("--public-base-url is required with --access public")
		}
		if _, err := core.ValidatePublicBaseURL(request.PublicBaseURL); err != nil {
			return publishExecutionResult{}, shared.UsageErrorf("--public-base-url: %v", err)
		}
		if request.PrivateOptionsExplicit {
			return publishExecutionResult{}, shared.UsageError("--url-ttl, --download-grace, and --download-endpoint are only valid with --access private")
		}
	} else if strings.TrimSpace(request.PublicBaseURL) != "" {
		return publishExecutionResult{}, shared.UsageError("--public-base-url is only valid with --access public")
	}
	if request.URLTTL <= 0 {
		return publishExecutionResult{}, shared.UsageError("--url-ttl must be positive")
	}
	if request.DownloadGrace < 0 {
		return publishExecutionResult{}, shared.UsageError("--download-grace must not be negative")
	}
	if accessMode == core.AccessPrivate && !validPrivatePublishLifetime(request.URLTTL, request.DownloadGrace) {
		return publishExecutionResult{}, shared.UsageError("--url-ttl plus --download-grace must not exceed 7d")
	}
	if request.VerifyTimeout <= 0 {
		return publishExecutionResult{}, shared.UsageError("--verify-timeout must be positive")
	}
	if request.ValidateOutput != nil {
		if err := request.ValidateOutput(); err != nil {
			return publishExecutionResult{}, err
		}
	}
	if _, err := core.NormalizePrefix(request.Prefix); err != nil {
		return publishExecutionResult{}, shared.UsageErrorf("--prefix: %v", err)
	}
	if !regionPattern.MatchString(request.Region) {
		return publishExecutionResult{}, shared.UsageError("--region must be 1-100 letters, digits, dots, underscores, or hyphens")
	}
	if !validPublishBucket(request.Bucket) {
		return publishExecutionResult{}, shared.UsageError("--bucket must be a bounded name without whitespace or control characters")
	}

	resolvedReceiptPath := strings.TrimSpace(request.ReceiptPath)
	resolvedLinkPath := strings.TrimSpace(request.LinkPath)
	if resolvedReceiptPath == "" || resolvedLinkPath == "" {
		return publishExecutionResult{}, shared.UsageError("--receipt and --link-path are required and must be outside --bundle-dir")
	}
	bundleRoot, err := rootfs.New(strings.TrimSpace(request.BundleDir))
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("distribute publish: open prepared bundle: %w", err)
	}
	defer bundleRoot.Close()
	if err := rejectBundleContainedArtifacts(bundleRoot, resolvedReceiptPath, resolvedLinkPath); err != nil {
		return publishExecutionResult{}, shared.UsageErrorf("publish artifacts: %v", err)
	}
	artifacts, err := inspectArtifactPaths(resolvedReceiptPath, resolvedLinkPath)
	if err != nil {
		return publishExecutionResult{}, shared.UsageErrorf("publish artifacts: %v", err)
	}
	defer artifacts.close()
	resolvedReceiptPath = artifacts.receiptPath
	resolvedLinkPath = artifacts.linkPath

	recoveredState, stateFound, err := artifacts.loadState()
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("distribute publish: %w", err)
	}
	if artifacts.receiptExists && !stateFound {
		return publishExecutionResult{}, fmt.Errorf("distribute publish: receipt exists without its sensitive link recovery artifact")
	}
	if !stateFound {
		if err := artifacts.preflightNew(); err != nil {
			return publishExecutionResult{}, shared.UsageErrorf("publish artifacts: %v", err)
		}
	}
	bundle, err := loadPreparedBundle(ctx, bundleRoot)
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("distribute publish: %w", err)
	}
	defer bundle.IPA.Close()
	if stateFound {
		if err := validateRecoveredState(recoveredState, bundle, validatedEndpoint.String(), effectiveDownloadEndpoint(request.Endpoint, request.DownloadEndpoint), normalizedPublicBase(request.PublicBaseURL), request.Region, request.AddressingStyle, request.Bucket, request.Prefix, accessMode, request.URLTTL, request.DownloadGrace, resolvedReceiptPath, resolvedLinkPath); err != nil {
			return publishExecutionResult{}, fmt.Errorf("distribute publish: %w", err)
		}
		verifier, err := newPublicationVerifier(request.VerifyTimeout)
		if err != nil {
			return publishExecutionResult{}, fmt.Errorf("configure publication verifier: %w", err)
		}
		if err := reverifyPublication(ctx, verifier, recoveredState.Receipt, recoveredState.Links, time.Now().UTC()); err != nil {
			return publishExecutionResult{}, fmt.Errorf("reverify recovered publication: %w", err)
		}
		if artifacts.receiptExists {
			if err := artifacts.verifyExactReceipt(recoveredState.Receipt); err != nil {
				return publishExecutionResult{}, fmt.Errorf("recover publish receipt: %w", err)
			}
		} else if err := artifacts.publishReceipt(recoveredState.Receipt); err != nil {
			return publishExecutionResult{}, fmt.Errorf("recover publish receipt: %w", err)
		}
		return publishExecutionResult{Receipt: recoveredState.Receipt, Recovered: true}, nil
	}

	staged, err := artifacts.stagePair()
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("stage local publish artifacts: %w", err)
	}
	defer staged.cleanup()

	setupCtx, setupCancel := shared.ContextWithUploadTimeout(ctx)
	store, credentialLimit, err := newObjectStore(setupCtx, core.S3StoreConfig{
		Endpoint: request.Endpoint, DownloadEndpoint: request.DownloadEndpoint, Region: request.Region,
		Bucket: request.Bucket, AddressingStyle: request.AddressingStyle,
	})
	setupCancel()
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("distribute publish: %w", err)
	}
	credentialSource := "standard-sdk-chain"
	if provider, ok := store.(interface{ CredentialSource() string }); ok {
		credentialSource = provider.CredentialSource()
	}
	fmt.Fprintf(diagnostic, "Publishing to endpoint=%s download-endpoint=%s bucket=%s region=%s addressing=%s prefix=%s access=%s credentials=%s\n", endpointOrigin(request.Endpoint), effectiveDownloadEndpoint(request.Endpoint, request.DownloadEndpoint), request.Bucket, request.Region, request.AddressingStyle, request.Prefix, accessMode, credentialSource)
	verifier, err := newPublicationVerifier(request.VerifyTimeout)
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("configure publication verifier: %w", err)
	}
	receipt, links, err := runPublish(ctx, bundle.IPA, bundle.Descriptor, core.PublishOptions{
		Store: publicationStore{delegate: store}, Verifier: verifier, Bucket: request.Bucket, Prefix: request.Prefix, Access: accessMode,
		PublicBaseURL: request.PublicBaseURL, URLTTL: request.URLTTL, DownloadGrace: request.DownloadGrace, CredentialLimit: credentialLimit,
	})
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("distribute publish: %w", err)
	}
	receipt.ReceiptPath = resolvedReceiptPath
	receipt.LinkPath = resolvedLinkPath
	receipt.Endpoint = validatedEndpoint.String()
	receipt.DownloadEndpoint = effectiveDownloadEndpoint(request.Endpoint, request.DownloadEndpoint)
	receipt.PublicBaseURL = normalizedPublicBase(request.PublicBaseURL)
	receipt.Region = request.Region
	receipt.AddressingStyle = request.AddressingStyle
	if accessMode == core.AccessPrivate {
		receipt.URLTTL = request.URLTTL.String()
		receipt.DownloadGrace = request.DownloadGrace.String()
	}

	state := publishState{SchemaVersion: "1", Receipt: receipt, Links: links}
	if err := staged.publish(state, receipt); err != nil {
		return publishExecutionResult{}, fmt.Errorf("write sensitive install link: %w", err)
	}
	return publishExecutionResult{Receipt: receipt}, nil
}

type privatePublishVerificationRequest struct {
	BundleDir     string
	ReceiptPath   string
	LinkPath      string
	VerifyTimeout time.Duration
}

// reverifyPrivatePublish verifies an existing private publication without
// creating directories, repairing receipts, refreshing links, or contacting
// the object store through any mutating operation.
func reverifyPrivatePublish(ctx context.Context, request privatePublishVerificationRequest) (publishExecutionResult, error) {
	if strings.TrimSpace(request.BundleDir) == "" || strings.TrimSpace(request.ReceiptPath) == "" || strings.TrimSpace(request.LinkPath) == "" {
		return publishExecutionResult{}, shared.UsageError("bundle directory, receipt, and link paths are required")
	}
	if request.VerifyTimeout <= 0 {
		return publishExecutionResult{}, shared.UsageError("verify timeout must be positive")
	}
	bundleDir := strings.TrimSpace(request.BundleDir)
	receiptPath := strings.TrimSpace(request.ReceiptPath)
	linkPath := strings.TrimSpace(request.LinkPath)
	bundleRoot, err := rootfs.New(bundleDir)
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("open prepared bundle: %w", err)
	}
	defer bundleRoot.Close()
	if err := rejectBundleContainedArtifacts(bundleRoot, receiptPath, linkPath); err != nil {
		return publishExecutionResult{}, shared.UsageErrorf("publish artifacts: %v", err)
	}
	artifacts, err := openExistingArtifactPaths(receiptPath, linkPath)
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("open publication artifacts: %w", err)
	}
	defer artifacts.close()
	state, found, err := artifacts.loadState()
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("load sensitive publication state: %w", err)
	}
	if !found {
		return publishExecutionResult{}, fmt.Errorf("sensitive publication state does not exist")
	}
	bundle, err := loadPreparedBundle(ctx, bundleRoot)
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("load prepared bundle: %w", err)
	}
	defer bundle.IPA.Close()
	if err := validateStoredPrivateState(state, bundle, artifacts.receiptPath, artifacts.linkPath); err != nil {
		return publishExecutionResult{}, fmt.Errorf("validate private publication state: %w", err)
	}
	if err := artifacts.verifyExactReceipt(state.Receipt); err != nil {
		return publishExecutionResult{}, fmt.Errorf("verify publication receipt: %w", err)
	}
	verifier, err := newPublicationVerifier(request.VerifyTimeout)
	if err != nil {
		return publishExecutionResult{}, fmt.Errorf("configure publication verifier: %w", err)
	}
	if err := reverifyPublication(ctx, verifier, state.Receipt, state.Links, time.Now().UTC()); err != nil {
		return publishExecutionResult{}, fmt.Errorf("reverify publication: %w", err)
	}
	return publishExecutionResult{Receipt: state.Receipt, Recovered: true}, nil
}

func openExistingArtifactPaths(receiptPath, linkPath string) (artifactPaths, error) {
	paths, err := anchorArtifactPaths(strings.TrimSpace(receiptPath), strings.TrimSpace(linkPath))
	if err != nil {
		return artifactPaths{}, err
	}

	var infos [2]os.FileInfo
	for index, item := range []struct {
		path  artifactPath
		label string
	}{{paths.receipt, "receipt"}, {paths.link, "sensitive link artifact"}} {
		info, found, err := inspectExistingProtectedPublishArtifact(item.path.root, item.path.relative, item.label)
		if err != nil {
			paths.close()
			return artifactPaths{}, err
		}
		if !found {
			paths.close()
			return artifactPaths{}, fmt.Errorf("%s does not exist", filepath.Join(item.path.rootPath, item.path.relative))
		}
		infos[index] = info
	}
	if os.SameFile(infos[0], infos[1]) {
		paths.close()
		return artifactPaths{}, fmt.Errorf("receipt and link paths resolve to the same physical destination")
	}
	paths.receiptExists = true
	paths.linkExists = true
	return paths, nil
}

func inspectExistingProtectedPublishArtifact(root *os.Root, name, label string) (os.FileInfo, bool, error) {
	file, err := secureopen.OpenExistingNoFollowInRoot(root, name)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	if err := validateProtectedPublishArtifact(file, info, label); err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func validateStoredPrivateState(state publishState, bundle *core.PreparedBundle, receiptPath, linkPath string) error {
	receipt := state.Receipt
	if receipt.Access != core.AccessPrivate || strings.TrimSpace(receipt.PublicBaseURL) != "" {
		return fmt.Errorf("publication is not private")
	}
	endpoint, err := core.ValidateEndpoint(receipt.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid saved endpoint")
	}
	downloadEndpoint, err := core.ValidateEndpoint(receipt.DownloadEndpoint)
	if err != nil {
		return fmt.Errorf("invalid saved download endpoint")
	}
	if receipt.AddressingStyle != "path" && receipt.AddressingStyle != "virtual" {
		return fmt.Errorf("invalid saved addressing style")
	}
	if !regionPattern.MatchString(receipt.Region) || !validPublishBucket(receipt.Bucket) {
		return fmt.Errorf("invalid saved object-store destination")
	}
	if _, err := core.NormalizePrefix(receipt.Prefix); err != nil {
		return fmt.Errorf("invalid saved object key prefix")
	}
	urlTTL, err := time.ParseDuration(receipt.URLTTL)
	if err != nil || urlTTL <= 0 {
		return fmt.Errorf("invalid saved URL lifetime")
	}
	downloadGrace, err := time.ParseDuration(receipt.DownloadGrace)
	if err != nil || !validPrivatePublishLifetime(urlTTL, downloadGrace) {
		return fmt.Errorf("invalid saved download grace")
	}
	return validateRecoveredState(
		state, bundle, endpoint.String(), downloadEndpoint.String(), "", receipt.Region, receipt.AddressingStyle,
		receipt.Bucket, receipt.Prefix, core.AccessPrivate, urlTTL, downloadGrace, receiptPath, linkPath,
	)
}

func validPublishBucket(bucket string) bool {
	return core.ValidateBucket(bucket) == nil
}

func validPrivatePublishLifetime(urlTTL, downloadGrace time.Duration) bool {
	const maximumPrivateLifetime = 7 * 24 * time.Hour
	return urlTTL > 0 &&
		downloadGrace >= 0 &&
		urlTTL <= maximumPrivateLifetime &&
		downloadGrace <= maximumPrivateLifetime-urlTTL
}
