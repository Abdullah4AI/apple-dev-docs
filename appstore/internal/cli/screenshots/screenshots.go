package screenshots

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/assets"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shots"
)

// ScreenshotsCommand returns the top-level screenshots command.
func ScreenshotsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("screenshots", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "screenshots",
		ShortUsage: "appstore screenshots <subcommand> [flags]",
		ShortHelp:  "Capture, frame, review (experimental local workflow), and upload App Store screenshots.",
		LongHelp: `Manage the full screenshot workflow from local capture to App Store upload.

Local screenshot automation commands are experimental.
If you face issues, please file feedback at:
https://github.com/Abdullah4AI/apple-developer-toolkit/appstore/issues/new/choose

Local workflow (experimental):
  appstore screenshots run --plan .appstore/screenshots.json
  appstore screenshots capture --bundle-id "com.example.app" --name home
  appstore screenshots frame --input ./screenshots/raw/home.png --device iphone-air
  appstore screenshots review-generate --framed-dir ./screenshots/framed
  appstore screenshots review-open --output-dir ./screenshots/review
  appstore screenshots review-approve --all-ready --output-dir ./screenshots/review
  appstore screenshots list-frame-devices --output json

App Store workflow:
  appstore screenshots list --version-localization "LOC_ID"
  appstore screenshots sizes --display-type "APP_IPHONE_69"
  appstore screenshots upload --version-localization "LOC_ID" --path "./screenshots" --device-type "IPHONE_69"
  appstore screenshots download --version-localization "LOC_ID" --output-dir "./screenshots/downloaded"
  appstore screenshots delete --id "SCREENSHOT_ID" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			shots.ShotsRunCommand(),
			shots.ShotsCaptureCommand(),
			shots.ShotsFrameCommand(),
			shots.ShotsFramesListDevicesCommand(),
			shots.ShotsReviewGenerateCommand(),
			shots.ShotsReviewOpenCommand(),
			shots.ShotsReviewApproveCommand(),
			assets.AssetsScreenshotsListCommand(),
			assets.AssetsScreenshotsSizesCommand(),
			assets.AssetsScreenshotsUploadCommand(),
			assets.AssetsScreenshotsDownloadCommand(),
			assets.AssetsScreenshotsDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
