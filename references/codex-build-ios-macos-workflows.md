# Codex Build iOS/macOS Apps Features in Apple Developer Toolkit CLI

Source inspected locally:

- `~/.codex/plugins/cache/openai-curated-remote/build-ios-apps/0.1.2/`
- `~/.codex/plugins/cache/openai-curated-remote/build-macos-apps/0.1.4/`

Use this reference when implementing or debugging the direct `appledev ios` and `appledev macos` command groups.

## What the Codex plugins provide

The Codex plugins are workflow bundles, not a hidden compiler. They provide:

1. Targeted iOS/macOS skill instructions.
2. Helper command/script patterns for Simulator previews, ETTrace, memgraph, and macOS run scripts.
3. For iOS, an MCP config:

```json
{
  "mcpServers": {
    "xcodebuildmcp": {
      "command": "npx",
      "args": ["-y", "xcodebuildmcp@latest", "mcp"],
      "env": {
        "XCODEBUILDMCP_ENABLED_WORKFLOWS": "simulator,ui-automation,debugging,logging"
      }
    }
  }
}
```

4. A proof loop: discover project → choose scheme/target → build/run → inspect UI/logs/performance evidence → patch → verify.

## CLI mapping

| Codex plugin feature | Direct CLI command |
|---|---|
| iOS debugger agent | `appledev ios doctor`, `appledev ios build`, `appledev ios logs`, `appledev ios screenshot` |
| iOS Simulator browser | `appledev ios mirror --sim <UDID>` |
| iOS ETTrace performance | `appledev ios ettrace` workflow contract |
| iOS memgraph leaks | `appledev ios memgraph --sim <UDID> --process <name>` |
| iOS App Intents | `appledev build` + `skills/ios-rules` references |
| SwiftUI Liquid Glass / patterns / refactor | `skills/swiftui-guides` + build/run verification |
| macOS build-run-debug | `appledev macos bootstrap`, `appledev macos run --mode ...` |
| macOS test triage | generated script + `swift test` / `xcodebuild test` |
| macOS signing entitlements | `appledev macos codesign-inspect --path <app-or-binary>` |
| SwiftPM macOS GUI app workflow | `appledev macos bootstrap --product <product> --app-name <name>` |
| macOS telemetry/logging | `appledev macos logs --process <name>`, `appledev macos run --mode telemetry` |

## iOS loop

Prefer `xcodebuildmcp` when the caller is an MCP-capable agent. For pure CLI use, fallback commands are available:

```bash
appledev ios doctor
appledev ios mcp-config
appledev ios simulators
appledev ios build --workspace App.xcworkspace --scheme App --device "iPhone 17 Pro"
appledev ios screenshot --sim <UDID> --output /tmp/app.png
appledev ios logs --sim <UDID> --process App
```

`serve-sim` is intentionally scoped by UDID:

```bash
appledev ios mirror --sim <UDID>
```

Never run unscoped simulator mirror cleanup because another task may own another simulator.

## macOS loop

For macOS apps, keep one stable project-local entrypoint:

```bash
appledev macos doctor
appledev macos bootstrap --app-name MyApp --scheme MyApp --project MyApp.xcodeproj
appledev macos run --mode verify
```

For SwiftPM GUI apps:

```bash
appledev macos bootstrap --app-name MyApp --product MyApp
appledev macos run --mode verify
```

The generated script stages `dist/<AppName>.app` for SwiftPM GUI products and launches it via `/usr/bin/open -n`, rather than raw-running GUI binaries.

## Guardrails

- Prefer read/list/status/validate/diff/dry-run before mutation.
- Do not mutate App Store Connect, signing, profiles, pricing, IAP, subscriptions, TestFlight, notarization, or release metadata without explicit approval.
- Do not store Apple credentials or `.p8` contents in the repo or generated scripts.
- Do not claim UI state, simulator state, leak fixes, or performance conclusions without tool output or artifacts.
