package xcodecloud

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

type xcodeCloudActionResourceGroupConfig struct {
	Name       string
	ShortUsage string
	ShortHelp  string
	LongHelp   string
	List       *ffcli.Command
	Get        *ffcli.Command
}

func newXcodeCloudActionResourceGroupCommand(config xcodeCloudActionResourceGroupConfig) *ffcli.Command {
	fs := flag.NewFlagSet(config.Name, flag.ExitOnError)

	return &ffcli.Command{
		Name:       config.Name,
		ShortUsage: config.ShortUsage,
		ShortHelp:  config.ShortHelp,
		LongHelp:   config.LongHelp,
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			config.List,
			config.Get,
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

type xcodeCloudActionResourceListConfig struct {
	ShortUsage  string
	ShortHelp   string
	LongHelp    string
	ParentUsage string
	ErrorPrefix string
	FetchPage   func(context.Context, *asc.Client, string, int, string) (asc.PaginatedResponse, error)
}

func newXcodeCloudActionResourceListCommand(config xcodeCloudActionResourceListConfig) *ffcli.Command {
	return shared.BuildPaginatedListCommand(shared.PaginatedListCommandConfig{
		FlagSetName: "list",
		Name:        "list",
		ShortUsage:  config.ShortUsage,
		ShortHelp:   config.ShortHelp,
		LongHelp:    config.LongHelp,
		ParentFlag:  "action-id",
		ParentUsage: config.ParentUsage,
		LimitMax:    200,
		ErrorPrefix: config.ErrorPrefix,
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return contextWithXcodeCloudTimeout(ctx, 0)
		},
		FetchPage: config.FetchPage,
	})
}

type xcodeCloudActionResourceGetConfig struct {
	ShortUsage  string
	ShortHelp   string
	LongHelp    string
	IDUsage     string
	ErrorPrefix string
	Fetch       func(context.Context, *asc.Client, string) (any, error)
}

func newXcodeCloudActionResourceGetCommand(config xcodeCloudActionResourceGetConfig) *ffcli.Command {
	return shared.BuildIDGetCommand(shared.IDGetCommandConfig{
		FlagSetName: "get",
		Name:        "get",
		ShortUsage:  config.ShortUsage,
		ShortHelp:   config.ShortHelp,
		LongHelp:    config.LongHelp,
		IDFlag:      "id",
		IDUsage:     config.IDUsage,
		ErrorPrefix: config.ErrorPrefix,
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return contextWithXcodeCloudTimeout(ctx, 0)
		},
		Fetch: config.Fetch,
	})
}

type xcodeCloudActionResourceCommandConfig struct {
	Name       string
	ShortUsage string
	ShortHelp  string
	LongHelp   string

	ListShortUsage  string
	ListShortHelp   string
	ListLongHelp    string
	ListParentUsage string
	ListErrorPrefix string
	ListFetchPage   func(context.Context, *asc.Client, string, int, string) (asc.PaginatedResponse, error)

	GetShortUsage  string
	GetShortHelp   string
	GetLongHelp    string
	GetIDUsage     string
	GetErrorPrefix string
	GetFetch       func(context.Context, *asc.Client, string) (any, error)
}

func newXcodeCloudActionResourceCommand(config xcodeCloudActionResourceCommandConfig) *ffcli.Command {
	return newXcodeCloudActionResourceGroupCommand(xcodeCloudActionResourceGroupConfig{
		Name:       config.Name,
		ShortUsage: config.ShortUsage,
		ShortHelp:  config.ShortHelp,
		LongHelp:   config.LongHelp,
		List: newXcodeCloudActionResourceListCommand(xcodeCloudActionResourceListConfig{
			ShortUsage:  config.ListShortUsage,
			ShortHelp:   config.ListShortHelp,
			LongHelp:    config.ListLongHelp,
			ParentUsage: config.ListParentUsage,
			ErrorPrefix: config.ListErrorPrefix,
			FetchPage:   config.ListFetchPage,
		}),
		Get: newXcodeCloudActionResourceGetCommand(xcodeCloudActionResourceGetConfig{
			ShortUsage:  config.GetShortUsage,
			ShortHelp:   config.GetShortHelp,
			LongHelp:    config.GetLongHelp,
			IDUsage:     config.GetIDUsage,
			ErrorPrefix: config.GetErrorPrefix,
			Fetch:       config.GetFetch,
		}),
	})
}

var xcodeCloudIssuesCommandConfig = xcodeCloudActionResourceCommandConfig{
	Name:       "issues",
	ShortUsage: "asc xcode-cloud issues <subcommand> [flags]",
	ShortHelp:  "List Xcode Cloud build issues.",
	LongHelp: `List Xcode Cloud build issues.

Examples:
  asc xcode-cloud issues list --action-id "ACTION_ID"
  asc xcode-cloud issues get --id "ISSUE_ID"`,
	ListShortUsage: "asc xcode-cloud issues list [flags]",
	ListShortHelp:  "List issues for a build action.",
	ListLongHelp: `List issues for a build action.

Examples:
  asc xcode-cloud issues list --action-id "ACTION_ID"
  asc xcode-cloud issues list --action-id "ACTION_ID" --output table
  asc xcode-cloud issues list --action-id "ACTION_ID" --limit 50
  asc xcode-cloud issues list --action-id "ACTION_ID" --paginate`,
	ListParentUsage: "Build action ID to list issues for",
	ListErrorPrefix: "xcode-cloud issues list",
	ListFetchPage: func(ctx context.Context, client *asc.Client, actionID string, limit int, next string) (asc.PaginatedResponse, error) {
		return client.GetCiBuildActionIssues(ctx, actionID,
			asc.WithCiIssuesLimit(limit),
			asc.WithCiIssuesNextURL(next),
		)
	},
	GetShortUsage: "asc xcode-cloud issues get --id \"ISSUE_ID\"",
	GetShortHelp:  "Get details for a build issue.",
	GetLongHelp: `Get details for a build issue.

Examples:
  asc xcode-cloud issues get --id "ISSUE_ID"
  asc xcode-cloud issues get --id "ISSUE_ID" --output table`,
	GetIDUsage:     "Issue ID",
	GetErrorPrefix: "xcode-cloud issues get",
	GetFetch: func(ctx context.Context, client *asc.Client, id string) (any, error) {
		return client.GetCiIssue(ctx, id)
	},
}

var xcodeCloudTestResultsCommandConfig = xcodeCloudActionResourceCommandConfig{
	Name:       "test-results",
	ShortUsage: "asc xcode-cloud test-results <subcommand> [flags]",
	ShortHelp:  "List Xcode Cloud test results.",
	LongHelp: `List Xcode Cloud test results.

Examples:
  asc xcode-cloud test-results list --action-id "ACTION_ID"
  asc xcode-cloud test-results get --id "TEST_RESULT_ID"`,
	ListShortUsage: "asc xcode-cloud test-results list [flags]",
	ListShortHelp:  "List test results for a build action.",
	ListLongHelp: `List test results for a build action.

Examples:
  asc xcode-cloud test-results list --action-id "ACTION_ID"
  asc xcode-cloud test-results list --action-id "ACTION_ID" --output table
  asc xcode-cloud test-results list --action-id "ACTION_ID" --limit 50
  asc xcode-cloud test-results list --action-id "ACTION_ID" --paginate`,
	ListParentUsage: "Build action ID to list test results for",
	ListErrorPrefix: "xcode-cloud test-results list",
	ListFetchPage: func(ctx context.Context, client *asc.Client, actionID string, limit int, next string) (asc.PaginatedResponse, error) {
		return client.GetCiBuildActionTestResults(ctx, actionID,
			asc.WithCiTestResultsLimit(limit),
			asc.WithCiTestResultsNextURL(next),
		)
	},
	GetShortUsage: "asc xcode-cloud test-results get --id \"TEST_RESULT_ID\"",
	GetShortHelp:  "Get details for a test result.",
	GetLongHelp: `Get details for a test result.

Examples:
  asc xcode-cloud test-results get --id "TEST_RESULT_ID"
  asc xcode-cloud test-results get --id "TEST_RESULT_ID" --output table`,
	GetIDUsage:     "Test result ID",
	GetErrorPrefix: "xcode-cloud test-results get",
	GetFetch: func(ctx context.Context, client *asc.Client, id string) (any, error) {
		return client.GetCiTestResult(ctx, id)
	},
}
