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

func workflowSubcommands(path []string, parent *endpointFlagValues) []*ffcli.Command {
	if len(path) == 1 && path[0] == "campaigns" {
		return []*ffcli.Command{
			campaignStatusWorkflowCommand("pause", "PAUSED", "Pause a campaign.", parent),
			campaignStatusWorkflowCommand("resume", "ENABLED", "Resume a campaign.", parent),
		}
	}
	return nil
}

func campaignStatusWorkflowCommand(name, status, shortHelp string, parent *endpointFlagValues) *ffcli.Command {
	fs := flag.NewFlagSet("campaigns "+name, flag.ExitOnError)
	flags := campaignStatusWorkflowFlags{
		common: commonFlags{
			AdsProfile: fs.String("ads-profile", "", "Use named Apple Ads authentication profile"),
			Org:        fs.String("org", "", "Apple Ads organization ID (or ASC_ADS_ORG_ID env)"),
		},
		output:   shared.BindOutputFlags(fs),
		flagSet:  fs,
		campaign: fs.String("campaign", "", "Apple Ads campaign ID (required)"),
		confirm:  fs.Bool("confirm", false, "Confirm this Apple Ads campaign status change"),
		parent:   parent,
	}
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "asc ads campaigns " + name + " [flags]",
		ShortHelp:  shortHelp,
		LongHelp: fmt.Sprintf(`%s

Endpoint: PUT v5/campaigns/{campaignId}

Examples:
  asc ads campaigns %s --campaign CAMPAIGN_ID --confirm --org ORG_ID`, shortHelp, name),
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := rejectUnexpectedArgs(args); err != nil {
				return err
			}
			return executeCampaignStatusWorkflow(ctx, name, status, flags)
		},
	}
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
		return fmt.Errorf("ads campaigns status workflow: missing campaigns update endpoint")
	}

	common, output := effectiveCampaignStatusWorkflowFlags(flags)
	outputFormat, err := shared.ValidateOutputFormat(*output.Output, *output.Pretty)
	if err != nil {
		return shared.UsageError(err.Error())
	}

	client, err := resolveClient(ctx, common, spec.RequiresOrg)
	if err != nil {
		return fmt.Errorf("ads campaigns %s: %w", commandName, err)
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
		return fmt.Errorf("ads campaigns %s: %w", commandName, err)
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
