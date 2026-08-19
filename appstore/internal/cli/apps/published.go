package apps

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const (
	appsPublishedAuditTimeout = 5 * time.Minute
	appsPublishedWorkers      = 4
)

var appsPublishedClientFactory = shared.GetASCClient

// PublishedApp describes an App Store Connect app with live published territory coverage.
type PublishedApp struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	BundleID                string `json:"bundleId"`
	SKU                     string `json:"sku"`
	PrimaryLocale           string `json:"primaryLocale,omitempty"`
	AvailabilityID          string `json:"availabilityId"`
	PublishedTerritoryCount int    `json:"publishedTerritoryCount"`
}

// AppsPublishedReport summarizes a complete account-wide published-app audit.
type AppsPublishedReport struct {
	AuditedAppCount   int            `json:"auditedAppCount"`
	PublishedAppCount int            `json:"publishedAppCount"`
	Apps              []PublishedApp `json:"apps"`
}

// AppsPublishedCommand returns the account-wide published-app audit command.
func AppsPublishedCommand() *ffcli.Command {
	fs := flag.NewFlagSet("apps published", flag.ExitOnError)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "published",
		ShortUsage: "asc apps published [flags]",
		ShortHelp:  "[experimental] List published apps and their published-territory counts.",
		LongHelp: `[experimental] List published apps and their published-territory counts.

This command is experimental.

The command audits every App Store Connect app record and all of its territory
availability pages. An app is published when at least one territory reports the
AVAILABLE content status. Apps without a published territory are omitted from
the result, while the report includes the total number of audited app records.

Examples:
  asc apps published
  asc apps published --output table
  asc apps published --output markdown`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("apps published does not accept positional arguments")
			}
			client, err := appsPublishedClientFactory()
			if err != nil {
				return fmt.Errorf("apps published: %w", err)
			}

			requestCtx, cancel := shared.ContextWithResolvedTimeout(ctx, appsPublishedAuditTimeout)
			defer cancel()
			appsResponse, err := fetchAllAppsForPublishedAudit(requestCtx, client)
			if err != nil {
				return fmt.Errorf("apps published: fetch apps: %w", err)
			}

			report, err := auditPublishedApps(requestCtx, client, appsResponse.Data)
			if err != nil {
				return fmt.Errorf("apps published: %w", err)
			}
			if strings.EqualFold(strings.TrimSpace(*output.Output), "json") {
				fmt.Fprintf(
					os.Stderr,
					"Audited %d app records; found %d published %s.\n",
					report.AuditedAppCount,
					report.PublishedAppCount,
					pluralizeApp(report.PublishedAppCount),
				)
			}
			return shared.PrintOutputWithRenderers(
				report,
				*output.Output,
				*output.Pretty,
				func() error { return renderAppsPublishedReport(report, false) },
				func() error { return renderAppsPublishedReport(report, true) },
			)
		},
	}
}

// fetchAllAppsForPublishedAudit retrieves every app record across all result pages.
func fetchAllAppsForPublishedAudit(ctx context.Context, client *asc.Client) (*asc.AppsResponse, error) {
	firstPage, err := client.GetApps(ctx, asc.WithAppsLimit(200), asc.WithAppsSort("name"))
	if err != nil {
		return nil, err
	}
	allPages, err := asc.PaginateAll(ctx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
		return client.GetApps(ctx, asc.WithAppsNextURL(nextURL))
	})
	if err != nil {
		return nil, err
	}
	response, ok := allPages.(*asc.AppsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected apps response type %T", allPages)
	}
	return response, nil
}

type publishedAppAuditResult struct {
	App            asc.Resource[asc.AppAttributes]
	AvailabilityID string
	TerritoryCount int
	Err            error
}

// auditPublishedApps audits app availability concurrently and returns published apps.
func auditPublishedApps(ctx context.Context, client *asc.Client, appResources []asc.Resource[asc.AppAttributes]) (AppsPublishedReport, error) {
	report := AppsPublishedReport{
		AuditedAppCount: len(appResources),
		Apps:            make([]PublishedApp, 0),
	}
	if len(appResources) == 0 {
		return report, nil
	}

	workerCount := min(appsPublishedWorkers, len(appResources))
	jobs := make(chan asc.Resource[asc.AppAttributes])
	results := make(chan publishedAppAuditResult, len(appResources))
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for app := range jobs {
				availabilityID, territoryCount, err := auditPublishedApp(ctx, client, app.ID)
				results <- publishedAppAuditResult{
					App:            app,
					AvailabilityID: availabilityID,
					TerritoryCount: territoryCount,
					Err:            err,
				}
			}
		}()
	}
	go func() {
		for _, app := range appResources {
			jobs <- app
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	failures := make([]string, 0)
	for result := range results {
		if result.Err != nil {
			failures = append(failures, fmt.Sprintf("%s (%s): %v", result.App.Attributes.Name, result.App.ID, result.Err))
			continue
		}
		if result.TerritoryCount == 0 {
			continue
		}
		report.Apps = append(report.Apps, PublishedApp{
			ID:                      result.App.ID,
			Name:                    result.App.Attributes.Name,
			BundleID:                result.App.Attributes.BundleID,
			SKU:                     result.App.Attributes.SKU,
			PrimaryLocale:           result.App.Attributes.PrimaryLocale,
			AvailabilityID:          result.AvailabilityID,
			PublishedTerritoryCount: result.TerritoryCount,
		})
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return AppsPublishedReport{}, fmt.Errorf("audit failed for %d of %d apps: %s", len(failures), len(appResources), strings.Join(failures, "; "))
	}

	sort.Slice(report.Apps, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(report.Apps[i].Name))
		right := strings.ToLower(strings.TrimSpace(report.Apps[j].Name))
		if left != right {
			return left < right
		}
		return report.Apps[i].ID < report.Apps[j].ID
	})
	report.PublishedAppCount = len(report.Apps)
	return report, nil
}

// auditPublishedApp returns an app's availability ID and published-territory count.
func auditPublishedApp(ctx context.Context, client *asc.Client, appID string) (string, int, error) {
	availability, err := client.GetAppAvailabilityV2(ctx, appID)
	if err != nil {
		if isPublishedAuditAvailabilityMissing(err) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("fetch availability: %w", err)
	}
	availabilityID := strings.TrimSpace(availability.Data.ID)
	if availabilityID == "" {
		return "", 0, fmt.Errorf("availability response did not include an ID")
	}

	firstPage, err := client.GetTerritoryAvailabilities(ctx, availabilityID, asc.WithTerritoryAvailabilitiesLimit(200))
	if err != nil {
		return availabilityID, 0, fmt.Errorf("fetch territory availabilities: %w", err)
	}
	allPages, err := asc.PaginateAll(ctx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
		return client.GetTerritoryAvailabilities(ctx, availabilityID, asc.WithTerritoryAvailabilitiesNextURL(nextURL))
	})
	if err != nil {
		return availabilityID, 0, fmt.Errorf("fetch territory availabilities: %w", err)
	}
	territories, ok := allPages.(*asc.TerritoryAvailabilitiesResponse)
	if !ok {
		return availabilityID, 0, fmt.Errorf("unexpected territory availabilities response type %T", allPages)
	}

	publishedTerritoryCount := 0
	for _, territory := range territories.Data {
		if hasAvailableContentStatus(territory.Attributes.ContentStatuses) {
			publishedTerritoryCount++
		}
	}
	return availabilityID, publishedTerritoryCount, nil
}

// isPublishedAuditAvailabilityMissing reports resource-specific missing availability.
func isPublishedAuditAvailabilityMissing(err error) bool {
	if err == nil || !errors.Is(err, asc.ErrNotFound) {
		return false
	}
	var apiErr *asc.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	description := strings.ToLower(strings.Join([]string{apiErr.Code, apiErr.Title, apiErr.Detail}, " "))
	return strings.Contains(description, "appavailabilities") ||
		strings.Contains(description, "app availability") ||
		strings.Contains(description, "appavailabilityv2")
}

// hasAvailableContentStatus reports whether any content status is AVAILABLE.
func hasAvailableContentStatus(statuses []string) bool {
	for _, status := range statuses {
		if strings.EqualFold(strings.TrimSpace(status), "AVAILABLE") {
			return true
		}
	}
	return false
}

// renderAppsPublishedReport writes table or Markdown rows and their audit totals.
func renderAppsPublishedReport(report AppsPublishedReport, markdown bool) error {
	headers := []string{"ID", "Name", "Bundle ID", "SKU", "Availability ID", "Published Territories"}
	rows := make([][]string, 0, len(report.Apps))
	for _, app := range report.Apps {
		rows = append(rows, []string{
			app.ID,
			app.Name,
			app.BundleID,
			app.SKU,
			app.AvailabilityID,
			strconv.Itoa(app.PublishedTerritoryCount),
		})
	}
	if markdown {
		asc.RenderMarkdown(headers, rows)
	} else {
		asc.RenderTable(headers, rows)
	}
	fmt.Fprintf(
		os.Stdout,
		"\nAudited %d app records; found %d published %s.\n",
		report.AuditedAppCount,
		report.PublishedAppCount,
		pluralizeApp(report.PublishedAppCount),
	)
	return nil
}

// pluralizeApp returns the singular or plural app label for a count.
func pluralizeApp(count int) string {
	if count == 1 {
		return "app"
	}
	return "apps"
}
