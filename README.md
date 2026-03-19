<div align="center">

<br>

# Apple Developer Toolkit

**Three tools, one binary. Docs, App Store, and app builder**

Search Apple docs, manage App Store Connect, and build multi-platform apps from natural language

<br>

[![Node.js](https://img.shields.io/badge/Node.js-18+-339933?style=flat&logo=node.js&logoColor=white)](https://nodejs.org)
[![Swift](https://img.shields.io/badge/Swift-6-FA7343?style=flat&logo=swift&logoColor=white)](https://swift.org)
[![macOS](https://img.shields.io/badge/macOS-only-000000?style=flat&logo=apple&logoColor=white)](https://developer.apple.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat)](LICENSE)

</div>

<br>

```
$ appledev build

> A habit tracker with streak counting and weekly grid

  ✓ Analyzed: StreakGrid
  ✓ Plan ready (11 files, 3 models)
  ✓ Build complete — 11 files
  ✓ Launched on iPhone 17 Pro
```

Ships as a single unified binary `appledev` with three independent tools. Each works on its own with different credential requirements.

## Install

```bash
brew install Abdullah4AI/tap/appledev
```

<details>
<summary>Install from source</summary>

```bash
git clone https://github.com/Abdullah4AI/apple-developer-toolkit.git
cd apple-developer-toolkit
bash scripts/setup.sh
```

</details>

## What's Inside

<table>
<tr>
<td align="center" width="33%">
<br>
<b>Documentation</b><br>
<sub>Apple docs + 1,267 WWDC sessions (2014-2025)</sub><br><br>
<code>node cli.js search</code>
</td>
<td align="center" width="33%">
<br>
<b>App Store Connect</b><br>
<sub>120+ commands for builds, TestFlight, submissions</sub><br><br>
<code>appledev store</code>
</td>
<td align="center" width="33%">
<br>
<b>App Builder</b><br>
<sub>Natural language to compiled SwiftUI apps</sub><br><br>
<code>appledev build</code>
</td>
</tr>
</table>

| Feature | Credentials | Works Without Setup |
|---|---|---|
| Documentation Search | None | Yes |
| App Store Connect | API key (.p8) | No |
| App Builder | LLM API key + Xcode | No |

## Documentation Search

Search Apple frameworks, symbols, and WWDC sessions locally. No API key needed.

```bash
node cli.js search "NavigationStack"        # Framework search
node cli.js symbols "UIView"                # Symbol lookup
node cli.js doc "/documentation/swiftui/view" # Full docs
node cli.js overview "SwiftUI"              # Framework overview
node cli.js samples "SwiftUI"               # Sample code
```

```bash
node cli.js wwdc-search "concurrency"       # Search WWDC talks
node cli.js wwdc-year 2025                  # Browse by year
node cli.js wwdc-topic "swiftui-ui-frameworks" # Browse by topic
```

## App Store Connect

120+ commands covering the full App Store Connect API.

```bash
# Authenticate
appledev store auth login --name "MyApp" \
  --key-id "KEY_ID" --issuer-id "ISSUER_ID" \
  --private-key ./AuthKey.p8

# Ship to TestFlight
appledev store publish testflight \
  --app "APP_ID" --ipa app.ipa --group "Beta" --wait

# Submit to App Store
appledev store publish appstore \
  --app "APP_ID" --ipa app.ipa --submit --confirm --wait

# Pre-submission validation
appledev store validate --app "APP_ID" --version-id "VER_ID"
```

<details>
<summary>All command categories</summary>

| Category | Commands |
|---|---|
| **Getting Started** | auth, doctor, init, docs |
| **Apps** | apps, app-setup, versions, localizations, screenshots, video-previews |
| **TestFlight** | testflight, builds, sandbox, feedback, crashes |
| **Review & Release** | review, reviews, submit, validate, publish, status |
| **Signing** | signing, bundle-ids, certificates, profiles, notarization |
| **Monetization** | iap, subscriptions, offer-codes, win-back-offers, pricing |
| **Analytics** | analytics, insights, finance, performance |
| **Automation** | xcode-cloud, webhooks, workflow, metadata, diff, migrate |

</details>

## App Builder

Build complete multi-platform Apple apps from natural language.

```
> A recipe app with ingredients list, step-by-step instructions, and a timer

  ✓ Analyzed: RecipeBook
  ✓ Plan ready (16 files, 4 models)
  ✓ Build complete — 16 files
  ✓ RecipeBook is ready!
```

### Platforms

<table>
<tr>
<td align="center"><img src="https://img.shields.io/badge/-iPhone-000?style=for-the-badge&logo=apple&logoColor=white" alt="iPhone"><br><sub>Default</sub></td>
<td align="center"><img src="https://img.shields.io/badge/-iPad-000?style=for-the-badge&logo=apple&logoColor=white" alt="iPad"><br><sub>"iPad"</sub></td>
<td align="center"><img src="https://img.shields.io/badge/-Apple%20Watch-000?style=for-the-badge&logo=apple&logoColor=white" alt="Apple Watch"><br><sub>"Apple Watch"</sub></td>
<td align="center"><img src="https://img.shields.io/badge/-Mac-000?style=for-the-badge&logo=apple&logoColor=white" alt="Mac"><br><sub>"Mac"</sub></td>
<td align="center"><img src="https://img.shields.io/badge/-Apple%20TV-000?style=for-the-badge&logo=apple&logoColor=white" alt="Apple TV"><br><sub>"Apple TV"</sub></td>
<td align="center"><img src="https://img.shields.io/badge/-Vision%20Pro-000?style=for-the-badge&logo=apple&logoColor=white" alt="Vision Pro"><br><sub>"Vision Pro"</sub></td>
</tr>
</table>

### How It Works

```
describe  →  analyze  →  plan  →  build  →  fix  →  run
   ↑            │          │        │        │       │
 prompt     app name    files    Swift    xcode-   iOS
            features    models   code     build    Simulator
```

| Phase | What happens |
|---|---|
| **Analyze** | Extracts app name, features, and core flow from your description |
| **Plan** | Produces a file-level build plan with models, navigation, and packages |
| **Build** | Generates Swift source, asset catalog, and Xcode project |
| **Fix** | Compiles with `xcodebuild` and auto-repairs until green |
| **Run** | Boots the simulator and launches the app |

### Commands

| Command | |
|---|---|
| `appledev build` | Interactive mode |
| `appledev build chat` | Edit an existing project |
| `appledev build fix` | Auto-fix compilation errors |
| `appledev build run` | Build and launch in simulator |
| `appledev build open` | Open in Xcode |
| `appledev build setup` | Install prerequisites |

<details>
<summary>Interactive commands</summary>

| Command | |
|---|---|
| `/run` | Build and launch in simulator |
| `/fix` | Auto-fix compilation errors |
| `/ask <question>` | Query your project (read-only) |
| `/open` | Open in Xcode |
| `/model <name>` | Switch AI model |
| `/info` | Show project info |
| `/usage` | Token usage and cost |

</details>

## Lifecycle Hooks

42 events across 4 categories. Get notified on Telegram, auto-distribute to TestFlight, git-tag releases, and chain operations into pipelines.

```bash
# Set up hooks with a template
bash scripts/hook-init.sh --template indie

# Fire a hook manually
bash scripts/hook-runner.sh build.done STATUS=success APP_NAME=MyApp

# Dry run
bash scripts/hook-runner.sh --dry-run build.done STATUS=success
```

### Templates

| Template | Focus |
|---|---|
| `indie` | Solo dev — Telegram notifications, auto TestFlight |
| `team` | Team — Slack + Telegram, git tagging, changelog |
| `ci` | CI/CD — Logging, test running, no interactive notifications |

### Built-in Scripts

| Script | Purpose |
|---|---|
| `notify-telegram.sh` | Send Telegram notification |
| `git-tag-release.sh` | Create and push git tag |
| `run-swift-tests.sh` | Run Swift tests |
| `generate-changelog.sh` | Generate changelog from git history |

<details>
<summary>Config example</summary>

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
      template: "{{if eq .STATUS \"success\"}}✅{{else}}❌{{end}} {{.APP_NAME}} build"

  store.review.approved:
    - name: tag-release
      run: "git tag v{{.VERSION}} && git push origin v{{.VERSION}}"
```

Config locations:
- **Global:** `~/.appledev/hooks.yaml`
- **Project:** `.appledev/hooks.yaml`

</details>

## AI Agent References

52 reference files for AI-assisted development:

| Reference | Count | Content |
|---|---|---|
| `references/ios-rules/` | 38 | iOS development rules (accessibility, dark mode, localization, etc.) |
| `references/swiftui-guides/` | 12 | SwiftUI best practices (Liquid Glass, navigation, state management, etc.) |
| `references/app-store-connect.md` | 1 | Complete CLI reference |
| `references/hooks-reference.md` | 1 | All hook events with context variables |

## Requirements

| Feature | Requires |
|---|---|
| Documentation | Node.js 18+ |
| App Store Connect | API key (.p8 file) |
| App Builder | Xcode + LLM API key |
| Hooks | Nothing |

## License

MIT

<div align="center">
<sub>Built by <a href="https://abdullah4.ai">Abdullah AlRashoudi</a></sub>
</div>
