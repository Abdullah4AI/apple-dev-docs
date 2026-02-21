# Apple Developer Tools

All-in-one Apple developer skill: documentation search, WWDC videos, and full App Store Connect management from the terminal. Built for AI agents and developers.

## Features

### Documentation & WWDC
- 1,267 WWDC sessions indexed locally (2014-2025)
- Direct integration with developer.apple.com
- Score-based search across all indexed sessions
- Zero external dependencies for docs

### App Store Connect
- **TestFlight** - builds, beta groups, testers, feedback, crash reports
- **Builds** - upload IPAs/PKGs, expire old builds, test notes, metrics
- **App Store** - versions, localizations, screenshots, review submissions, phased releases
- **Signing** - certificates, provisioning profiles, bundle IDs, capabilities
- **Subscriptions & IAP** - create and manage subscriptions, in-app purchases, offer codes, pricing
- **Analytics & Sales** - download sales reports, analytics data, finance reports
- **Xcode Cloud** - trigger workflows, monitor build runs, download artifacts
- **Notarization** - submit, poll, and retrieve logs for macOS notarization
- **Game Center** - achievements, leaderboards, leaderboard sets, localizations
- **Screenshots & Previews** - capture, frame, and manage App Store media
- **Webhooks** - create and manage App Store Connect webhooks
- **Workflow** - multi-step automation via workflow.json
- **Publish** - end-to-end TestFlight and App Store submission workflows
- **Validate** - pre-submission checks for metadata, screenshots, age ratings
- **Migrate** - Fastlane compatibility (import/export metadata)

## Requirements

- **Node.js** v18+ (for documentation search)
- App Store Connect CLI (auto-installed via `bash scripts/setup.sh`) for App Store Connect

## Installation

For AI agents (Codex, Claude Code, etc.):

```bash
npx skills add Abdullah4AI/apple-dev-docs
```

For ClawHub:

```bash
clawhub install apple-dev-docs
```

## Quick Start

### Documentation

```bash
# Search documentation
node cli.js search "SwiftUI animation"

# Search WWDC sessions
node cli.js wwdc-search "swift concurrency"

# Get video details with transcript
node cli.js wwdc-video 2024-10169

# Explore a framework
node cli.js overview "SwiftUI"
```

### App Store Connect

```bash
# Auto-install App Store Connect CLI (first run only)
bash scripts/setup.sh

# Setup authentication
appstore auth login --name "MyApp" --key-id "KEY_ID" --issuer-id "ISSUER_ID" --private-key /path/to/AuthKey.p8

# List your apps
appstore apps

# Upload and distribute to TestFlight
appstore publish testflight --app "APP_ID" --ipa "app.ipa" --group "Beta Testers" --wait

# Submit to App Store
appstore publish appstore --app "APP_ID" --ipa "app.ipa" --submit --confirm --wait

# Check reviews
appstore reviews --app "APP_ID" --output table

# Download sales report
appstore analytics sales --vendor "VENDOR" --type SALES --subtype SUMMARY --frequency DAILY --date "2024-01-20"
```

If installed as a skill, just ask your AI agent naturally:

> "Search Apple docs for SwiftUI animations"
> "Upload my build to TestFlight"
> "List my provisioning profiles"
> "Show me crash reports for my app"

## Documentation Commands

| Command | Description |
|---------|-------------|
| `search "query"` | Search Apple Developer Documentation |
| `symbols "UIView"` | Search framework classes, structs, protocols |
| `doc "/path/to/doc"` | Get detailed documentation by path |
| `apis "UIViewController"` | Find related APIs |
| `platform "UIScrollView"` | Check platform/version compatibility |
| `similar "UIPickerView"` | Find recommended alternatives |
| `tech` | List all Apple technologies |
| `overview "SwiftUI"` | Get technology overview guide |
| `samples "SwiftUI"` | Find sample code projects |
| `updates` | Latest documentation updates |

## WWDC Commands

| Command | Description |
|---------|-------------|
| `wwdc-search "async"` | Search WWDC sessions |
| `wwdc-video 2024-100` | Get video details and transcript |
| `wwdc-topics` | List 20 topic categories |
| `wwdc-topic "swift"` | List videos by topic |
| `wwdc-years` | List years with video counts |
| `wwdc-year 2025` | List all videos for a year |

## App Store Connect Commands

| Task | Command |
|------|---------|
| List apps | `appstore apps` |
| List builds | `appstore builds list --app "APP_ID"` |
| Upload build | `appstore builds upload --app "APP_ID" --ipa "app.ipa" --wait` |
| Latest build | `appstore builds latest --app "APP_ID"` |
| Expire old builds | `appstore builds expire-all --app "APP_ID" --older-than 90d --confirm` |
| TestFlight groups | `appstore testflight beta-groups list --app "APP_ID"` |
| Add tester | `appstore testflight beta-testers add --app "APP_ID" --email "t@test.com" --group "Beta"` |
| Publish TestFlight | `appstore publish testflight --app "APP_ID" --ipa "app.ipa" --group "Beta" --wait` |
| Submit App Store | `appstore publish appstore --app "APP_ID" --ipa "app.ipa" --submit --confirm --wait` |
| Certificates | `appstore certificates list` |
| Profiles | `appstore profiles list` |
| Create version | `appstore versions create --app "APP_ID" --version "1.0.0"` |
| Reviews | `appstore reviews --app "APP_ID" --output table` |
| Sales report | `appstore analytics sales --vendor "VENDOR" --type SALES --subtype SUMMARY --frequency DAILY --date "2024-01-20"` |
| Xcode Cloud | `appstore xcode-cloud run --app "APP_ID" --workflow "CI" --branch "main" --wait` |
| Notarize | `appstore notarization submit --file ./MyApp.zip --wait` |
| Validate | `appstore validate --app "APP_ID" --version-id "VERSION_ID" --strict` |
| Subscriptions | `appstore subscriptions groups list --app "APP_ID"` |
| IAP | `appstore iap list --app "APP_ID"` |
| Game Center | `appstore game-center achievements list --app "APP_ID"` |
| Webhooks | `appstore webhooks list --app "APP_ID"` |

For the complete App Store Connect reference with all commands and flags, see [references/app-store-connect.md](references/app-store-connect.md).

## Options

| Option | Description |
|--------|-------------|
| `--limit <n>` | Limit number of results |
| `--output table` | Human-readable table output |
| `--output markdown` | Markdown format |
| `--paginate` | Fetch all pages |
| `--sort field` | Sort results (prefix `-` for desc) |
| `--confirm` | Required for destructive operations |

## WWDC Index

Pre-built index of 1,267 WWDC sessions. To rebuild:

```bash
node build-wwdc-index.js
```

### Coverage

| Year | Sessions |
|------|----------|
| 2025 | 122 |
| 2024 | 123 |
| 2023 | 181 |
| 2022 | 185 |
| 2021 | 207 |
| 2020 | 209 |
| 2019 | 153 |
| 2018 | 18 |
| 2017 | 36 |
| 2016 | 16 |
| 2015 | 11 |
| 2014 | 6 |

## How It Works

Documentation searches query developer.apple.com directly. WWDC data is indexed locally for fast offline search. App Store Connect operations use the built-in CLI which communicates with the App Store Connect API using your API key credentials.

## License

MIT
