package ads

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/appleads"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

type campaignStatusWorkflowFlags struct {
	common   commonFlags
	output   shared.OutputFlags
	flagSet  *flag.FlagSet
	campaign *string
	confirm  *bool
	parent   *endpointFlagValues
}

type platformCampaignStatusWorkflowFlags struct {
	common   commonFlags
	output   shared.OutputFlags
	campaign *string
	confirm  *bool
}

func workflowSubcommands(path []string, parent *endpointFlagValues) []*ffcli.Command {
	if len(path) == 1 && path[0] == "campaigns" {
		return []*ffcli.Command{
			campaignStatusWorkflowCommand("pause", "PAUSED", "Pause a campaign.", parent),
			campaignStatusWorkflowCommand("resume", "ENABLED", "Resume a campaign.", parent),
		}
	}
	return nil
}

func platformWorkflowSubcommands(path []string) []*ffcli.Command {
	if len(path) != 1 || path[0] != "campaigns" {
		return nil
	}
	return []*ffcli.Command{
		platformCampaignStatusWorkflowCommand("pause", "PAUSED", "Pause a Platform API campaign."),
		platformCampaignStatusWorkflowCommand("resume", "ENABLED", "Resume a Platform API campaign."),
	}
}

func platformCampaignStatusWorkflowCommand(name, status, shortHelp string) *ffcli.Command {
	fs := flag.NewFlagSet("campaigns "+name, flag.ExitOnError)
	flags := platformCampaignStatusWorkflowFlags{
		common: commonFlags{
			AdsProfile: fs.String("ads-profile", "", "Use named Apple Ads authentication profile"),
			AdAccount:  fs.String("ad-account", "", "Apple Ads ad account ID (or ASC_ADS_AD_ACCOUNT_ID env)"),
		},
		output:   bindAdsRawOutputFlags(fs),
		campaign: fs.String("campaign", "", "Apple Ads Platform campaign ID (required)"),
		confirm:  fs.Bool("confirm", false, "Confirm this Apple Ads Platform campaign status change"),
	}
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "asc ads campaigns " + name + " [flags]",
		ShortHelp:  shortHelp,
		LongHelp:   platformCampaignStatusWorkflowHelp(name, status, shortHelp),
		FlagSet:    fs,
		UsageFunc:  shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := rejectUnexpectedArgs(args); err != nil {
				return err
			}
			return executePlatformCampaignStatusWorkflow(ctx, name, status, flags)
		},
	}
}

func executePlatformCampaignStatusWorkflow(ctx context.Context, commandName, status string, flags platformCampaignStatusWorkflowFlags) error {
	campaignID := value(flags.campaign)
	if campaignID == "" {
		return shared.UsageError("--campaign is required")
	}
	if status != "PAUSED" && (flags.confirm == nil || !*flags.confirm) {
		return shared.UsageError("--confirm is required")
	}

	spec, ok := appleads.PlatformEndpointByCommandPath("campaigns", "update")
	if !ok {
		return fmt.Errorf("ads campaigns status workflow: missing campaigns update endpoint")
	}
	outputFormat, err := validateAdsRawOutput(flags.output)
	if err != nil {
		return shared.UsageError(err.Error())
	}
	client, _, err := resolvePlatformClientAndAdAccountID(ctx, flags.common, spec.Context)
	if err != nil {
		return fmt.Errorf("ads campaigns %s: %w", commandName, err)
	}

	body, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return err
	}
	requestCtx, cancel := requestContext(ctx)
	defer cancel()
	result, err := client.Do(requestCtx, spec, map[string]string{"id": campaignID}, nil, body)
	if err != nil {
		return fmt.Errorf("ads campaigns %s: %w", commandName, err)
	}
	return shared.PrintOutput(result, outputFormat, *flags.output.Pretty)
}

func platformCampaignStatusWorkflowHelp(name, status, shortHelp string) string {
	confirm := ""
	if status != "PAUSED" {
		confirm = " --confirm"
	}
	return fmt.Sprintf(`%s

Endpoint: PUT v1/campaigns/{id}
Payload: {"status":"%s"}

Pausing is a spend-reducing safety operation and does not require --confirm;
resuming requires --confirm.

Examples:
  asc ads campaigns %s --campaign CAMPAIGN_ID%s --ad-account AD_ACCOUNT_ID`, shortHelp, status, name, confirm)
}

func campaignStatusWorkflowCommand(name, status, shortHelp string, parent *endpointFlagValues) *ffcli.Command {
	fs := flag.NewFlagSet("v5 campaigns "+name, flag.ExitOnError)
	flags := campaignStatusWorkflowFlags{
		common: commonFlags{
			AdsProfile: fs.String("ads-profile", "", "Use named Apple Ads authentication profile"),
			Org:        fs.String("org", "", "Apple Ads organization ID (or ASC_ADS_ORG_ID env)"),
		},
		output:   bindAdsRawOutputFlags(fs),
		flagSet:  fs,
		campaign: fs.String("campaign", "", "Apple Ads campaign ID (required)"),
		confirm:  fs.Bool("confirm", false, "Confirm this Apple Ads campaign status change"),
		parent:   parent,
	}
	command := &ffcli.Command{
		Name:       name,
		ShortUsage: "asc ads v5 campaigns " + name + " [flags]",
		ShortHelp:  shortHelp,
		LongHelp: fmt.Sprintf(`%s

Endpoint: PUT v5/campaigns/{campaignId}

Examples:
  asc ads v5 campaigns %s --campaign CAMPAIGN_ID --confirm --org ORG_ID`, shortHelp, name),
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := rejectUnexpectedArgs(args); err != nil {
				return err
			}
			return executeCampaignStatusWorkflow(ctx, name, status, flags)
		},
	}
	return markAdsLegacyCommandDeprecated(command, []string{"v5", "campaigns", name}, adsLegacyMigration{
		kind:        adsLegacyBreaking,
		replacement: []string{"campaigns", name},
	})
}

func executeCampaignStatusWorkflow(ctx context.Context, commandName, status string, flags campaignStatusWorkflowFlags) error {
	campaignID := value(flags.campaign)
	if campaignID == "" {
		return shared.UsageError("--campaign is required")
	}
	if err := validateAdsIntegerFlag("campaign", campaignID); err != nil {
		return err
	}
	if flags.confirm == nil || !*flags.confirm {
		return shared.UsageError("--confirm is required")
	}

	spec, ok := appleads.EndpointByCommandPath("campaigns", "update")
	if !ok {
		return fmt.Errorf("ads v5 campaigns status workflow: missing campaigns update endpoint")
	}

	common, output := effectiveCampaignStatusWorkflowFlags(flags)
	outputFormat, err := validateAdsRawOutput(output)
	if err != nil {
		return shared.UsageError(err.Error())
	}

	client, err := resolveClient(ctx, common, spec.RequiresOrg)
	if err != nil {
		return fmt.Errorf("ads v5 campaigns %s: %w", commandName, err)
	}

	requestCtx, cancel := requestContext(ctx)
	defer cancel()

	body, err := json.Marshal(map[string]map[string]string{
		"campaign": {"status": status},
	})
	if err != nil {
		return err
	}
	result, err := client.Do(requestCtx, spec, map[string]string{"campaignId": campaignID}, nil, body)
	if err != nil {
		return fmt.Errorf("ads v5 campaigns %s: %w", commandName, err)
	}
	return shared.PrintOutput(result, outputFormat, *output.Pretty)
}

func effectiveCampaignStatusWorkflowFlags(flags campaignStatusWorkflowFlags) (commonFlags, shared.OutputFlags) {
	common := flags.common
	output := flags.output
	if flags.parent == nil || flags.parent.flagSet == nil {
		return common, output
	}
	if !flagWasSet(flags.flagSet, "ads-profile") && flagWasSet(flags.parent.flagSet, "ads-profile") {
		common.AdsProfile = flags.parent.common.AdsProfile
	}
	if !flagWasSet(flags.flagSet, "org") && flagWasSet(flags.parent.flagSet, "org") {
		common.Org = flags.parent.common.Org
	}
	if !flagWasSet(flags.flagSet, "output") && flagWasSet(flags.parent.flagSet, "output") {
		output.Output = flags.parent.output.Output
	}
	if !flagWasSet(flags.flagSet, "pretty") && flagWasSet(flags.parent.flagSet, "pretty") {
		output.Pretty = flags.parent.output.Pretty
	}
	return common, output
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	if fs == nil {
		return false
	}
	wasSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}

func validateAdsIntegerFlag(name, raw string) error {
	spec := appleads.EndpointSpec{
		PathParams: []appleads.ParamSpec{{
			Name:     name,
			Flag:     name,
			Type:     appleads.ParamInt,
			Required: true,
		}},
	}
	flags := endpointFlagValues{
		pathStrings: map[string]*string{name: &raw},
	}
	_, err := collectPathParams(spec, flags)
	if err != nil {
		return shared.UsageError(err.Error())
	}
	return nil
}
