package shared

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

const (
	bulkAvailabilityTimeout = 5 * time.Minute
	bulkAvailabilityWorkers = 4
)

var availabilityClientFactory = getASCClient

// AvailabilitySetCommandConfig configures the availability set command.
type AvailabilitySetCommandConfig struct {
	FlagSetName                      string
	CommandName                      string
	ShortUsage                       string
	ShortHelp                        string
	LongHelp                         string
	ErrorPrefix                      string
	IncludeAvailableInNewTerritories bool
}

// AvailabilityRemoveFromSaleCommandConfig configures the remove-from-sale command.
type AvailabilityRemoveFromSaleCommandConfig struct {
	ClientFactory func() (*asc.Client, error)
}

// AvailabilityRemoveFromSaleResult summarizes a verified remove-from-sale operation.
type AvailabilityRemoveFromSaleResult struct {
	AppID                          string   `json:"appId"`
	AvailabilityID                 string   `json:"availabilityId"`
	Status                         string   `json:"status"`
	AvailableInNewTerritories      bool     `json:"availableInNewTerritories"`
	TotalTerritories               int      `json:"totalTerritories"`
	UpdatedTerritories             int      `json:"updatedTerritories"`
	AlreadyUnavailableTerritories  int      `json:"alreadyUnavailableTerritories"`
	VerifiedUnavailableTerritories int      `json:"verifiedUnavailableTerritories"`
	FailedTerritories              []string `json:"failedTerritories"`
}

// NewAvailabilitySetCommand builds a shared availability set command.
func NewAvailabilitySetCommand(config AvailabilitySetCommandConfig) *ffcli.Command {
	fs := flag.NewFlagSet(config.FlagSetName, flag.ExitOnError)

	appID := fs.String("app", "", "App Store Connect app ID (or ASC_APP_ID)")
	territory := BindOnceCSVFlag(fs, "territory", "Territory inputs (comma-separated; accepts alpha-2, alpha-3, or exact English country names, e.g., US,USA,France)")
	allTerritories := fs.Bool("all-territories", false, "Apply to all territories (overrides --territory)")
	var available OptionalBool
	fs.Var(&available, "available", "Set availability: true or false")
	var availableInNewTerritories OptionalBool
	if config.IncludeAvailableInNewTerritories {
		fs.Var(&availableInNewTerritories, "available-in-new-territories", "Verify the existing new-territory policy (optional; this API cannot change it): true or false")
	}
	output := BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       config.CommandName,
		ShortUsage: config.ShortUsage,
		ShortHelp:  config.ShortHelp,
		LongHelp:   config.LongHelp,
		FlagSet:    fs,
		UsageFunc:  DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			resolvedAppID := resolveAppID(*appID)
			if resolvedAppID == "" {
				fmt.Fprintln(os.Stderr, "Error: --app is required (or set ASC_APP_ID)")
				return MissingRequiredUsageError("--app")
			}
			if !*allTerritories && strings.TrimSpace(territory.String()) == "" {
				fmt.Fprintln(os.Stderr, "Error: --territory or --all-territories is required")
				return MissingRequiredUsageError("")
			}
			if !available.IsSet() {
				fmt.Fprintln(os.Stderr, "Error: --available is required (true or false)")
				return MissingRequiredUsageError("--available")
			}
			var territories []string
			if !*allTerritories {
				normalizedTerritories, normalizeErr := normalizeASCTerritoryCSV(territory.String())
				if normalizeErr != nil {
					return UsageError(normalizeErr.Error())
				}
				territories = normalizedTerritories
				if len(territories) == 0 {
					fmt.Fprintln(os.Stderr, "Error: --territory must include at least one value")
					return flag.ErrHelp
				}
			}

			availableValue := available.Value()

			client, err := availabilityClientFactory()
			if err != nil {
				return fmt.Errorf("%s: %w", config.ErrorPrefix, err)
			}

			requestCtx, cancel := contextWithAvailabilityTimeout(ctx, *allTerritories)
			defer cancel()

			var expectedAvailableInNewTerritories *bool
			if config.IncludeAvailableInNewTerritories && availableInNewTerritories.IsSet() {
				value := availableInNewTerritories.Value()
				expectedAvailableInNewTerritories = &value
			}
			resp, _, err := executeTerritoryAvailabilityUpdate(requestCtx, client, availabilityUpdateRequest{
				AppID:                             resolvedAppID,
				Territories:                       territories,
				AllTerritories:                    *allTerritories,
				Available:                         availableValue,
				ExpectedAvailableInNewTerritories: expectedAvailableInNewTerritories,
				ErrorPrefix:                       config.ErrorPrefix,
			})
			if err != nil {
				return err
			}
			return printOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

// NewAvailabilityRemoveFromSaleCommand builds a command that makes every
// existing territory unavailable while preserving the app's new-territory policy.
func NewAvailabilityRemoveFromSaleCommand(config AvailabilityRemoveFromSaleCommandConfig) *ffcli.Command {
	fs := flag.NewFlagSet("pricing availability remove-from-sale", flag.ExitOnError)
	appID := fs.String("app", "", "App Store Connect app ID (or ASC_APP_ID)")
	confirm := fs.Bool("confirm", false, "Confirm removal from sale in all current territories")
	output := BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "remove-from-sale",
		ShortUsage: "asc pricing availability remove-from-sale --app \"APP_ID\" --confirm",
		ShortHelp:  "Remove an app from sale in all current territories.",
		LongHelp: `Remove an app from sale in all current territories.

This command uses the public App Store Connect API. It makes every existing
territory unavailable and verifies the final state. It does not delete the app
record, and it preserves the existing availableInNewTerritories policy because
Apple does not expose an update operation for that setting.

Examples:
  asc pricing availability remove-from-sale --app "123456789" --confirm`,
		FlagSet:   fs,
		UsageFunc: DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return UsageError("pricing availability remove-from-sale does not accept positional arguments")
			}
			if !*confirm {
				fmt.Fprintln(os.Stderr, "Error: --confirm is required")
				return MissingRequiredUsageError("--confirm")
			}
			resolvedAppID := resolveAppID(*appID)
			if resolvedAppID == "" {
				fmt.Fprintln(os.Stderr, "Error: --app is required (or set ASC_APP_ID)")
				return MissingRequiredUsageError("--app")
			}
			if config.ClientFactory == nil {
				return fmt.Errorf("pricing availability remove-from-sale: client factory is not configured")
			}
			client, err := config.ClientFactory()
			if err != nil {
				return fmt.Errorf("pricing availability remove-from-sale: %w", err)
			}

			requestCtx, cancel := contextWithAvailabilityTimeout(ctx, true)
			defer cancel()
			_, summary, updateErr := executeTerritoryAvailabilityUpdate(requestCtx, client, availabilityUpdateRequest{
				AppID:          resolvedAppID,
				AllTerritories: true,
				Available:      false,
				ErrorPrefix:    "pricing availability remove-from-sale",
			})
			if updateErr != nil && len(summary.FailedTerritories) == 0 {
				return updateErr
			}

			status := "removedFromSale"
			if updateErr != nil {
				status = "partialFailure"
			}
			result := AvailabilityRemoveFromSaleResult{
				AppID:                          resolvedAppID,
				AvailabilityID:                 summary.AvailabilityID,
				Status:                         status,
				AvailableInNewTerritories:      summary.AvailableInNewTerritories,
				TotalTerritories:               summary.TotalTerritories,
				UpdatedTerritories:             summary.UpdatedTerritories,
				AlreadyUnavailableTerritories:  summary.SkippedTerritories,
				VerifiedUnavailableTerritories: summary.VerifiedTerritories,
				FailedTerritories:              append([]string{}, summary.FailedTerritories...),
			}
			if updateErr != nil {
				fmt.Fprintf(
					os.Stderr,
					"App %s removal is incomplete: %d of %d current territories are verified unavailable; preserved availableInNewTerritories=%t.\n",
					resolvedAppID,
					result.VerifiedUnavailableTerritories,
					result.TotalTerritories,
					result.AvailableInNewTerritories,
				)
			} else {
				fmt.Fprintf(
					os.Stderr,
					"App %s is unavailable in all %d current territories; preserved availableInNewTerritories=%t.\n",
					resolvedAppID,
					result.VerifiedUnavailableTerritories,
					result.AvailableInNewTerritories,
				)
			}
			if result.AvailableInNewTerritories {
				fmt.Fprintln(os.Stderr, "Warning: Apple may automatically enable future App Store territories under the preserved policy.")
			}
			renderErr := PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error {
					renderAvailabilityRemoveFromSaleResult(result, false)
					return nil
				},
				func() error {
					renderAvailabilityRemoveFromSaleResult(result, true)
					return nil
				},
			)
			return errors.Join(updateErr, renderErr)
		},
	}
}

type availabilityUpdateRequest struct {
	AppID                             string
	Territories                       []string
	AllTerritories                    bool
	Available                         bool
	ExpectedAvailableInNewTerritories *bool
	ErrorPrefix                       string
}

type availabilityUpdateSummary struct {
	AvailabilityID            string
	AvailableInNewTerritories bool
	TotalTerritories          int
	UpdatedTerritories        int
	SkippedTerritories        int
	VerifiedTerritories       int
	FailedTerritories         []string
}

func executeTerritoryAvailabilityUpdate(ctx context.Context, client *asc.Client, request availabilityUpdateRequest) (*asc.AppAvailabilityV2Response, availabilityUpdateSummary, error) {
	summary := availabilityUpdateSummary{}
	resp, err := client.GetAppAvailabilityV2(ctx, request.AppID)
	if err != nil {
		if isAppAvailabilityMissing(err) {
			return nil, summary, NewErrorWithCause(
				fmt.Errorf(
					"%s: app availability not found for app %q; this command only updates existing app availability, so use \"asc pricing availability create\" first. If Apple rejects public-API bootstrap, authenticate with \"asc web auth login --apple-id EMAIL\" and use \"asc web apps availability create\", or configure Pricing and Availability in App Store Connect: %w",
					request.ErrorPrefix,
					request.AppID,
					asc.ErrNotFound,
				),
				err,
			)
		}
		return nil, summary, fmt.Errorf("%s: %w", request.ErrorPrefix, err)
	}
	availabilityID := strings.TrimSpace(resp.Data.ID)
	if availabilityID == "" {
		return nil, summary, fmt.Errorf("%s: app availability ID missing from response", request.ErrorPrefix)
	}
	summary.AvailabilityID = availabilityID
	summary.AvailableInNewTerritories = resp.Data.Attributes.AvailableInNewTerritories

	if request.ExpectedAvailableInNewTerritories != nil && resp.Data.Attributes.AvailableInNewTerritories != *request.ExpectedAvailableInNewTerritories {
		return nil, summary, fmt.Errorf(
			"%s: --available-in-new-territories does not match the existing policy (current value: %t); the public API cannot change this setting",
			request.ErrorPrefix,
			resp.Data.Attributes.AvailableInNewTerritories,
		)
	}

	territoryResp, err := getAllTerritoryAvailabilities(ctx, client, availabilityID)
	if err != nil {
		return nil, summary, fmt.Errorf("%s: %w", request.ErrorPrefix, err)
	}
	territoryMap, err := mapTerritoryAvailabilities(territoryResp)
	if err != nil {
		return nil, summary, fmt.Errorf("%s: %w", request.ErrorPrefix, err)
	}

	var targets []availabilityEditTarget
	if request.AllTerritories {
		territoryIDs := make([]string, 0, len(territoryMap))
		for territoryID := range territoryMap {
			territoryIDs = append(territoryIDs, territoryID)
		}
		sort.Strings(territoryIDs)
		targets = make([]availabilityEditTarget, 0, len(territoryIDs))
		for _, territoryID := range territoryIDs {
			availability := territoryMap[territoryID]
			targets = append(targets, availabilityEditTarget{TerritoryID: territoryID, AvailabilityID: availability.ID, Available: availability.Attributes.Available})
		}
	} else {
		missingTerritories := make([]string, 0)
		targets = make([]availabilityEditTarget, 0, len(request.Territories))
		for _, territoryID := range request.Territories {
			availability, ok := territoryMap[territoryID]
			if !ok {
				missingTerritories = append(missingTerritories, territoryID)
				continue
			}
			targets = append(targets, availabilityEditTarget{TerritoryID: territoryID, AvailabilityID: availability.ID, Available: availability.Attributes.Available})
		}
		if len(missingTerritories) > 0 {
			return nil, summary, fmt.Errorf("%s: territory availability not found for territories: %s", request.ErrorPrefix, strings.Join(missingTerritories, ", "))
		}
	}

	summary.TotalTerritories = len(targets)
	pending := make([]availabilityEditTarget, 0, len(targets))
	for _, target := range targets {
		if target.Available == request.Available {
			summary.SkippedTerritories++
			continue
		}
		pending = append(pending, target)
	}
	if len(pending) == 0 {
		summary.VerifiedTerritories = summary.SkippedTerritories
		fmt.Fprintf(os.Stderr, "Updated 0 territories; %d already matched.\n", summary.SkippedTerritories)
		return resp, summary, nil
	}

	fmt.Fprintf(os.Stderr, "Updating availability for %d territories (%d already matched)...\n", len(pending), summary.SkippedTerritories)
	patchErrors := updateTerritoryAvailabilityTargets(ctx, client, pending, request.Available)
	verifiedResp, err := getAllTerritoryAvailabilities(ctx, client, availabilityID)
	if err != nil {
		return nil, summary, fmt.Errorf(
			"%s: attempted %d territory updates (%d request failures, %d skipped); final verification failed: %w",
			request.ErrorPrefix,
			len(pending),
			len(patchErrors),
			summary.SkippedTerritories,
			err,
		)
	}
	verifiedMap, err := mapTerritoryAvailabilities(verifiedResp)
	if err != nil {
		return nil, summary, fmt.Errorf("%s: verify territory availabilities: %w", request.ErrorPrefix, err)
	}

	failureDetails := make([]string, 0)
	for _, target := range targets {
		verified, ok := verifiedMap[target.TerritoryID]
		if ok && verified.Attributes.Available == request.Available {
			if target.Available != request.Available {
				summary.UpdatedTerritories++
			}
			continue
		}
		summary.FailedTerritories = append(summary.FailedTerritories, target.TerritoryID)
		if patchErr := patchErrors[target.TerritoryID]; patchErr != nil {
			failureDetails = append(failureDetails, fmt.Sprintf("%s: %v", target.TerritoryID, patchErr))
		} else if !ok {
			failureDetails = append(failureDetails, fmt.Sprintf("%s: missing from verification response", target.TerritoryID))
		} else if target.Available == request.Available {
			failureDetails = append(failureDetails, fmt.Sprintf("%s: state changed during verification", target.TerritoryID))
		} else {
			failureDetails = append(failureDetails, fmt.Sprintf("%s: requested state was not observed", target.TerritoryID))
		}
	}

	summary.VerifiedTerritories = len(targets) - len(summary.FailedTerritories)
	if len(summary.FailedTerritories) > 0 {
		sort.Strings(summary.FailedTerritories)
		sort.Strings(failureDetails)
		return nil, summary, fmt.Errorf(
			"%s: updated %d, skipped %d, failed %d (%s): %s",
			request.ErrorPrefix,
			summary.UpdatedTerritories,
			summary.SkippedTerritories,
			len(summary.FailedTerritories),
			strings.Join(summary.FailedTerritories, ", "),
			strings.Join(failureDetails, "; "),
		)
	}

	fmt.Fprintf(os.Stderr, "Updated %d territories; %d already matched; verified %d updated territories.\n", summary.UpdatedTerritories, summary.SkippedTerritories, len(pending))
	return resp, summary, nil
}

func renderAvailabilityRemoveFromSaleResult(result AvailabilityRemoveFromSaleResult, markdown bool) {
	headers := []string{"Field", "Value"}
	rows := [][]string{
		{"App ID", result.AppID},
		{"Availability ID", result.AvailabilityID},
		{"Status", result.Status},
		{"Available in new territories", fmt.Sprintf("%t", result.AvailableInNewTerritories)},
		{"Total territories", fmt.Sprintf("%d", result.TotalTerritories)},
		{"Updated territories", fmt.Sprintf("%d", result.UpdatedTerritories)},
		{"Already unavailable", fmt.Sprintf("%d", result.AlreadyUnavailableTerritories)},
		{"Verified unavailable", fmt.Sprintf("%d", result.VerifiedUnavailableTerritories)},
		{"Failed territories", strings.Join(result.FailedTerritories, ", ")},
	}
	if markdown {
		asc.RenderMarkdown(headers, rows)
		return
	}
	asc.RenderTable(headers, rows)
}

func contextWithAvailabilityTimeout(ctx context.Context, allTerritories bool) (context.Context, context.CancelFunc) {
	if allTerritories {
		return ContextWithResolvedTimeout(ctx, bulkAvailabilityTimeout)
	}
	return contextWithTimeout(ctx)
}

type availabilityEditTarget struct {
	TerritoryID    string
	AvailabilityID string
	Available      bool
}

type territoryAvailabilityUpdateResult struct {
	TerritoryID string
	Err         error
}

func updateTerritoryAvailabilityTargets(ctx context.Context, client *asc.Client, targets []availabilityEditTarget, available bool) map[string]error {
	workerCount := min(bulkAvailabilityWorkers, len(targets))
	jobs := make(chan availabilityEditTarget)
	results := make(chan territoryAvailabilityUpdateResult, len(targets))

	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for target := range jobs {
				_, err := client.UpdateTerritoryAvailability(ctx, target.AvailabilityID, asc.TerritoryAvailabilityUpdateAttributes{
					Available: &available,
				})
				results <- territoryAvailabilityUpdateResult{TerritoryID: target.TerritoryID, Err: err}
			}
		}()
	}

	go func() {
		for _, target := range targets {
			jobs <- target
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	errs := make(map[string]error)
	for result := range results {
		if result.Err != nil {
			errs[result.TerritoryID] = result.Err
		}
	}
	return errs
}

func getAllTerritoryAvailabilities(ctx context.Context, client *asc.Client, availabilityID string) (*asc.TerritoryAvailabilitiesResponse, error) {
	firstPage, err := client.GetTerritoryAvailabilities(ctx, availabilityID, asc.WithTerritoryAvailabilitiesLimit(200))
	if err != nil {
		return nil, err
	}
	paginated, err := asc.PaginateAll(ctx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
		return client.GetTerritoryAvailabilities(ctx, availabilityID, asc.WithTerritoryAvailabilitiesNextURL(nextURL))
	})
	if err != nil {
		return nil, err
	}
	resp, ok := paginated.(*asc.TerritoryAvailabilitiesResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected territory availabilities response")
	}
	return resp, nil
}

type territoryAvailabilityIDPayload struct {
	Territory string `json:"t"`
}

// MapTerritoryAvailabilityIDs maps territory IDs to territory-availability IDs.
func MapTerritoryAvailabilityIDs(resp *asc.TerritoryAvailabilitiesResponse) (map[string]string, error) {
	availabilities, err := mapTerritoryAvailabilities(resp)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]string, len(availabilities))
	for territoryID, availability := range availabilities {
		ids[territoryID] = availability.ID
	}
	return ids, nil
}

func mapTerritoryAvailabilities(resp *asc.TerritoryAvailabilitiesResponse) (map[string]asc.Resource[asc.TerritoryAvailabilityAttributes], error) {
	if resp == nil {
		return nil, fmt.Errorf("territory availabilities response is nil")
	}
	availabilities := make(map[string]asc.Resource[asc.TerritoryAvailabilityAttributes], len(resp.Data))
	for _, item := range resp.Data {
		territoryID := ""
		if len(item.Relationships) > 0 {
			var relationships asc.TerritoryAvailabilityRelationships
			if err := json.Unmarshal(item.Relationships, &relationships); err != nil {
				return nil, fmt.Errorf("decode territory availability relationships for %q: %w", item.ID, err)
			}
			territoryID = strings.ToUpper(strings.TrimSpace(relationships.Territory.Data.ID))
		}
		if territoryID == "" {
			var ok bool
			territoryID, ok = territoryIDFromAvailabilityID(item.ID)
			if !ok {
				return nil, fmt.Errorf("territory availability %q missing territory id", item.ID)
			}
		}
		availabilities[territoryID] = item
	}
	return availabilities, nil
}

func territoryIDFromAvailabilityID(availabilityID string) (string, bool) {
	trimmed := strings.TrimSpace(availabilityID)
	if trimmed == "" {
		return "", false
	}
	decoded, err := base64.RawStdEncoding.DecodeString(trimmed)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			decoded, err = base64.RawURLEncoding.DecodeString(trimmed)
			if err != nil {
				decoded, err = base64.URLEncoding.DecodeString(trimmed)
				if err != nil {
					return "", false
				}
			}
		}
	}
	var payload territoryAvailabilityIDPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return "", false
	}
	territoryID := strings.TrimSpace(payload.Territory)
	if territoryID == "" {
		return "", false
	}
	return strings.ToUpper(territoryID), true
}
