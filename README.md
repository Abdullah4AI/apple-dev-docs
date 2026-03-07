# Apple Developer Toolkit

All-in-one Apple developer CLI: documentation search, WWDC videos, App Store Connect management, autonomous iOS app builder, and lifecycle hooks with Telegram notifications. Ships as a **single unified binary** (`appledev`).

## Install

```bash
brew install Abdullah4AI/tap/appledev
```

```bash
clawhub install apple-developer-toolkit
```

## Quick Start

```bash
appledev build                    # Build an iOS app from a description
appledev store apps               # List your App Store Connect apps
appledev hooks init               # Set up lifecycle hooks
appledev notify telegram --message "Hello"  # Send a Telegram notification
node cli.js search "NavigationStack"        # Search Apple docs
```

## What's Inside

| Tool | Command | Description |
|------|---------|-------------|
| **Docs** | `node cli.js` | Apple docs + 1,267 WWDC sessions (2014-2025) |
| **Store** | `appledev store` | 120+ App Store Connect commands |
| **Builder** | `appledev build` | AI-powered iOS/macOS/watchOS/tvOS/visionOS app generation |
| **Hooks** | `appledev hooks` | Lifecycle hooks with Telegram/Slack notifications |

## Lifecycle Hooks

Hooks fire automatically when you build, upload, submit, or release. Get Telegram notifications, auto-distribute to TestFlight, git-tag releases, and chain operations into pipelines.

### Setup

```bash
appledev hooks init --template indie    # Creates ~/.appledev/hooks.yaml
```

### 31 Events

| Category | Events |
|----------|--------|
| **Build** | `build.start`, `build.compile.success`, `build.compile.failure`, `build.fix.start`, `build.fix.done`, `build.run.success`, `build.done` |
| **Store** | `store.upload.start/done/failure`, `store.processing.done`, `store.testflight.distribute`, `store.submit.start/done/failure`, `store.review.approved/rejected`, `store.release.done`, `store.validate.pass/fail` |
| **Pipeline** | `pipeline.start`, `pipeline.step.done`, `pipeline.done`, `pipeline.failure` |

### Config Example

```yaml
version: 1
notifiers:
  telegram:
    enabled: true
    bot_token_keychain: "my-bot-token"
    chat_id: "123456"

hooks:
  build.done:
    - name: notify-build
      notify: telegram
      template: "{{if eq .STATUS \"success\"}}✅{{else}}❌{{end}} {{.APP_NAME}} build {{.STATUS}}"
      when: always

  store.upload.done:
    - name: auto-testflight
      run: "appledev store publish testflight --app {{.APP_ID}} --build {{.BUILD_ID}} --group Beta"
      when: success

  store.review.approved:
    - name: tag-release
      run: "git tag v{{.VERSION}} && git push origin v{{.VERSION}}"
    - name: notify
      notify: telegram
      template: "🎉 {{.APP_NAME}} v{{.VERSION}} approved"
```

### Templates

| Template | For | Includes |
|----------|-----|----------|
| `indie` | Solo developers | Telegram notifications + auto TestFlight |
| `team` | Teams | Slack + Telegram, git tagging, changelog |
| `ci` | CI/CD pipelines | Logging, test running, no interactive notifications |

### CLI Commands

```bash
appledev hooks init [--template indie|team|ci] [--project]
appledev hooks list [--event "store.*"]
appledev hooks fire <event> [KEY=VALUE...]
appledev hooks fire --dry-run build.done STATUS=success
appledev hooks validate
appledev notify telegram --message "Deploy done"
appledev notify slack --webhook $URL --message "Build ready"
```

### Config Locations

| Scope | Path | Behavior |
|-------|------|----------|
| Global | `~/.appledev/hooks.yaml` | Applies to all projects |
| Project | `.appledev/hooks.yaml` | Extends/overrides global |

## Documentation Search

Search Apple Developer Documentation and WWDC sessions locally. No API key needed.

```bash
node cli.js search "NavigationStack"
node cli.js symbols "UIView"
node cli.js doc "/documentation/swiftui/navigationstack"
node cli.js overview "SwiftUI"
node cli.js samples "SwiftUI"
node cli.js wwdc-search "concurrency"
node cli.js wwdc-year 2025
node cli.js wwdc-topic "swiftui-ui-frameworks"
```

## App Store Connect

120+ commands covering the entire App Store Connect API.

### Auth

```bash
appledev store auth login --name "MyApp" --key-id "KEY_ID" --issuer-id "ISSUER_ID" --private-key /path/to/AuthKey.p8
```

### Common Workflows

```bash
# List apps
appledev store apps

# Upload and distribute to TestFlight
appledev store publish testflight --app "APP_ID" --ipa "app.ipa" --group "Beta" --wait

# Submit for App Store review
appledev store publish appstore --app "APP_ID" --ipa "app.ipa" --submit --confirm --wait

# Pre-submission validation
appledev store validate --app "APP_ID" --version-id "VER_ID" --strict

# Release pipeline dashboard
appledev store status --app "APP_ID" --output table

# Analytics
appledev store insights weekly --app "APP_ID" --source analytics

# Multi-step workflow automation
appledev store workflow run beta BUILD_ID:123 GROUP_ID:abc
```

### Full Command Reference

| Category | Commands |
|----------|----------|
| Getting Started | auth, doctor, init, docs |
| Apps | apps, app-setup, app-tags, app-info, app-infos, versions, localizations, screenshots, video-previews |
| TestFlight | testflight, builds, build-bundles, pre-release-versions, sandbox, feedback, crashes |
| Review & Release | review, reviews, submit, validate, publish, status |
| Signing | signing, bundle-ids, certificates, profiles, merchant-ids, pass-type-ids, notarization |
| Monetization | iap, subscriptions, offer-codes, win-back-offers, promoted-purchases, app-events, pricing, pre-orders |
| Analytics | analytics, insights, finance, performance |
| Automation | xcode-cloud, webhooks, notify, workflow, metadata, diff, migrate |
| Team | account, users, actors, devices |

## iOS App Builder

Build complete multi-platform Apple apps from natural language. Powered by Claude Code.

```bash
appledev build                     # Describe your app and build it
appledev build chat                # Edit an existing app interactively
appledev build fix                 # Auto-fix compilation errors
appledev build run                 # Build and launch in simulator
appledev build open                # Open in Xcode
appledev build setup               # Install prerequisites
```

### Supported Platforms

iOS, iPadOS, macOS, watchOS, tvOS, visionOS

### How It Works

```
describe → analyze → plan → build → fix → run
```

1. **Analyze** - Extracts app name, features, platform from your description
2. **Plan** - Produces file-level build plan with data models and navigation
3. **Build** - Generates Swift/SwiftUI source files and project config
4. **Fix** - Compiles and auto-repairs until build succeeds
5. **Run** - Boots simulator and launches the app

## Reference Files

50+ reference files for AI agents building Apple apps:

| Reference | Count | Content |
|-----------|-------|---------|
| `references/ios-rules/` | 38 files | iOS development rules (accessibility, app review, gestures, haptics, etc.) |
| `references/swiftui-guides/` | 12 files | SwiftUI best practices (Liquid Glass, navigation, state management, animations, etc.) |
| `references/app-store-connect.md` | 1 file | Complete App Store Connect CLI reference |
| `references/hooks-reference.md` | 1 file | All 31 hook events with context variables |

## Requirements

| Feature | Requires |
|---------|----------|
| Documentation search | Node.js 18+ |
| App Store Connect | API key (.p8 file) |
| iOS app builder | Xcode + LLM API key |
| Hooks | Nothing (works out of the box) |

## Building from Source

```bash
git clone https://github.com/Abdullah4AI/apple-developer-toolkit.git
cd apple-developer-toolkit
bash scripts/setup.sh
```

## License

MIT
