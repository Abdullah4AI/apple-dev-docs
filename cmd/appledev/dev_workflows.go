package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type commandCheck struct {
	Name     string
	Required bool
	Purpose  string
}

func newIOSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ios",
		Aliases: []string{"iphone", "ipad", "sim"},
		Short:   "Build iOS Apps plugin workflows: Simulator, UI automation, logs, ETTrace, memgraph",
		Long: `Direct CLI surface for the OpenAI Codex Build iOS Apps plugin workflows.

The command group exposes the plugin's practical features without requiring Codex:
- xcodebuildmcp setup/doctor helpers
- Simulator discovery, screenshots, logs, and browser mirroring
- xcodebuild/xcrun fallback build-run loops
- memgraph and ETTrace workflow entrypoints
- feature catalog matching the plugin skill bundle`,
	}
	cmd.AddCommand(iosFeaturesCommand())
	cmd.AddCommand(iosDoctorCommand())
	cmd.AddCommand(iosMCPConfigCommand())
	cmd.AddCommand(iosSimulatorsCommand())
	cmd.AddCommand(iosBuildCommand())
	cmd.AddCommand(iosScreenshotCommand())
	cmd.AddCommand(iosLogsCommand())
	cmd.AddCommand(iosMirrorCommand())
	cmd.AddCommand(iosMemgraphCommand())
	cmd.AddCommand(iosETTraceCommand())
	return cmd
}

func newMacOSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "macos",
		Aliases: []string{"mac", "desktop"},
		Short:   "Build macOS Apps plugin workflows: build/run scripts, logs, signing, packaging prep",
		Long: `Direct CLI surface for the OpenAI Codex Build macOS Apps plugin workflows.

The command group exposes the plugin's practical features without requiring Codex:
- project-local script/build_and_run.sh bootstrap
- shell-first run/debug/log/telemetry/verify loops
- SwiftPM GUI .app bundling guidance
- signing, entitlements, and Gatekeeper inspection
- feature catalog matching the plugin skill bundle`,
	}
	cmd.AddCommand(macosFeaturesCommand())
	cmd.AddCommand(macosDoctorCommand())
	cmd.AddCommand(macosBootstrapCommand())
	cmd.AddCommand(macosRunCommand())
	cmd.AddCommand(macosLogsCommand())
	cmd.AddCommand(macosCodesignInspectCommand())
	return cmd
}

func iosFeaturesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "features",
		Short: "List Build iOS Apps plugin features now available in appledev",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), `Build iOS Apps features in appledev ios:

Workflow / Codex skill                 CLI surface
- ios-debugger-agent                   appledev ios doctor | build | logs | screenshot
- ios-simulator-browser                appledev ios mirror --sim <UDID>
- ios-ettrace-performance              appledev ios ettrace --help
- ios-memgraph-leaks                   appledev ios memgraph --sim <UDID> --process <name>
- ios-app-intents                      covered by appledev build + skills/ios-rules references
- swiftui-liquid-glass                 covered by skills/swiftui-guides + docs search
- swiftui-performance-audit            covered by ettrace/memgraph + build verification
- swiftui-ui-patterns                  covered by skills/swiftui-guides references
- swiftui-view-refactor                covered by build verification + references

Recommended proof loop:
  appledev ios doctor
  appledev ios simulators
  appledev ios build --workspace App.xcworkspace --scheme App --device "iPhone 17 Pro"
  appledev ios screenshot --sim <UDID> --output /tmp/app.png
`)
			return nil
		},
	}
}

func macosFeaturesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "features",
		Short: "List Build macOS Apps plugin features now available in appledev",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), `Build macOS Apps features in appledev macos:

Workflow / Codex skill                 CLI surface
- build-run-debug                      appledev macos bootstrap | run --mode run/debug/logs/telemetry/verify
- test-triage                          use generated script plus swift test / xcodebuild test
- signing-entitlements                 appledev macos codesign-inspect --path <app-or-binary>
- swiftpm-macos                        bootstrap supports SwiftPM products and .app staging
- packaging-notarization               store/notarization commands + signing inspection guardrails
- swiftui-patterns                     covered by skills/swiftui-guides references
- liquid-glass                         covered by docs search + SwiftUI references
- window-management                    covered by macOS references and run verification
- appkit-interop                       covered by macOS references and build verification
- view-refactor                        covered by build/test/run loop
- telemetry                            appledev macos run --mode telemetry | logs --process <name>

Recommended proof loop:
  appledev macos doctor
  appledev macos bootstrap --app-name MyApp --scheme MyApp
  appledev macos run --mode verify
`)
			return nil
		},
	}
}

func iosDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check iOS workflow prerequisites",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printChecks(cmd, []commandCheck{
				{"xcodebuild", true, "build iOS projects"},
				{"xcrun", true, "Simulator control through simctl"},
				{"npx", false, "run xcodebuildmcp and serve-sim"},
				{"instruments", false, "advanced Instruments/trace workflows"},
			})
		},
	}
}

func macosDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check macOS workflow prerequisites",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printChecks(cmd, []commandCheck{
				{"xcodebuild", false, "build Xcode macOS projects"},
				{"swift", false, "build SwiftPM projects"},
				{"lldb", false, "debug macOS apps"},
				{"codesign", false, "inspect signing and entitlements"},
				{"spctl", false, "Gatekeeper assessment"},
				{"plutil", false, "inspect Info.plist and entitlements"},
				{"log", false, "stream unified logs"},
			})
		},
	}
}

func iosMCPConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp-config",
		Short: "Print the xcodebuildmcp config used by Build iOS Apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), `{
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
`)
			return nil
		},
	}
}

func iosSimulatorsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "simulators",
		Short: "List available iOS Simulators",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStreaming(cmd, "xcrun", "simctl", "list", "devices", "available")
		},
	}
}

func iosBuildCommand() *cobra.Command {
	var workspace, project, scheme, device, configuration string
	var run bool
	c := &cobra.Command{
		Use:   "build",
		Short: "Build an iOS app for Simulator using xcodebuild fallback workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			if scheme == "" {
				return errors.New("--scheme is required")
			}
			dest := "platform=iOS Simulator"
			if device != "" {
				dest += ",name=" + device
			}
			buildArgs := []string{}
			if workspace != "" {
				buildArgs = append(buildArgs, "-workspace", workspace)
			} else if project != "" {
				buildArgs = append(buildArgs, "-project", project)
			} else {
				return errors.New("--workspace or --project is required")
			}
			buildArgs = append(buildArgs, "-scheme", scheme, "-configuration", configuration, "-destination", dest, "build")
			if err := runStreaming(cmd, "xcodebuild", buildArgs...); err != nil {
				return err
			}
			if run {
				fmt.Fprintln(cmd.OutOrStdout(), "\nBuild succeeded. To install/launch, use xcodebuildmcp build_run_sim in an agent session or xcrun simctl install/launch with the built .app path.")
			}
			return nil
		},
	}
	c.Flags().StringVar(&workspace, "workspace", "", "Path to .xcworkspace")
	c.Flags().StringVar(&project, "project", "", "Path to .xcodeproj")
	c.Flags().StringVar(&scheme, "scheme", "", "Xcode scheme")
	c.Flags().StringVar(&device, "device", "", "Simulator device name, e.g. iPhone 17 Pro")
	c.Flags().StringVar(&configuration, "configuration", "Debug", "Build configuration")
	c.Flags().BoolVar(&run, "run-note", false, "Print install/launch follow-up note after build")
	return c
}

func iosScreenshotCommand() *cobra.Command {
	var sim, output string
	c := &cobra.Command{
		Use:   "screenshot",
		Short: "Capture a Simulator screenshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sim == "" {
				return errors.New("--sim UDID is required")
			}
			if output == "" {
				output = filepath.Join(os.TempDir(), "appledev-sim-screenshot.png")
			}
			return runStreaming(cmd, "xcrun", "simctl", "io", sim, "screenshot", output)
		},
	}
	c.Flags().StringVar(&sim, "sim", "", "Simulator UDID")
	c.Flags().StringVar(&output, "output", "", "Output PNG path")
	return c
}

func iosLogsCommand() *cobra.Command {
	var sim, process string
	c := &cobra.Command{
		Use:   "logs",
		Short: "Stream Simulator logs for a process",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sim == "" || process == "" {
				return errors.New("--sim UDID and --process are required")
			}
			predicate := fmt.Sprintf(`process == "%s"`, process)
			return runStreaming(cmd, "xcrun", "simctl", "spawn", sim, "log", "stream", "--style", "compact", "--predicate", predicate)
		},
	}
	c.Flags().StringVar(&sim, "sim", "", "Simulator UDID")
	c.Flags().StringVar(&process, "process", "", "App process name")
	return c
}

func iosMirrorCommand() *cobra.Command {
	var sim string
	c := &cobra.Command{
		Use:   "mirror",
		Short: "Mirror one Simulator in the browser using serve-sim",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sim == "" {
				return errors.New("--sim UDID is required; unscoped serve-sim is intentionally not allowed")
			}
			if err := runStreaming(cmd, "npx", "--yes", "serve-sim@latest", "--kill", sim); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "serve-sim cleanup warning: %v\n", err)
			}
			return runStreaming(cmd, "npx", "--yes", "serve-sim@latest", sim)
		},
	}
	c.Flags().StringVar(&sim, "sim", "", "Simulator UDID")
	return c
}

func iosMemgraphCommand() *cobra.Command {
	var sim, process, output string
	c := &cobra.Command{
		Use:   "memgraph",
		Short: "Capture a memgraph from a running Simulator process",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sim == "" || process == "" {
				return errors.New("--sim UDID and --process are required")
			}
			if output == "" {
				output = filepath.Join(os.TempDir(), process+".memgraph")
			}
			return runStreaming(cmd, "xcrun", "simctl", "spawn", sim, "xcrun", "memgraph", process, output)
		},
	}
	c.Flags().StringVar(&sim, "sim", "", "Simulator UDID")
	c.Flags().StringVar(&process, "process", "", "App process name or PID inside Simulator")
	c.Flags().StringVar(&output, "output", "", "Output .memgraph path")
	return c
}

func iosETTraceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ettrace",
		Short: "Print the ETTrace profiling workflow contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), `ETTrace workflow:
1. Capture one focused launch/runtime flow at a time.
2. Temporarily link ETTrace only into the exact Simulator Debug target.
3. Build and collect UUID-matched app/framework dSYMs.
4. Treat unsymbolicated first-party frames as failed evidence.
5. Preserve fresh output_*.json immediately; ignore stale viewer output.json.
6. Use analyzer output as evidence, then remove temporary ETTrace wiring unless intentionally kept.

This command documents the workflow; project-specific ETTrace wiring remains explicit so the CLI does not mutate app targets silently.
`)
			return nil
		},
	}
}

func macosBootstrapCommand() *cobra.Command {
	var appName, scheme, workspace, project, product string
	c := &cobra.Command{
		Use:   "bootstrap",
		Short: "Create/update script/build_and_run.sh for macOS app workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appName == "" {
				appName = inferAppName(scheme, product)
			}
			if appName == "" {
				return errors.New("--app-name is required when --scheme or --product is not provided")
			}
			scriptPath := filepath.Join("script", "build_and_run.sh")
			if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
				return err
			}
			content := macBuildScript(appName, scheme, workspace, project, product)
			if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\nRun: appledev macos run --mode verify\n", scriptPath)
			return nil
		},
	}
	c.Flags().StringVar(&appName, "app-name", "", "macOS app/process name")
	c.Flags().StringVar(&scheme, "scheme", "", "Xcode scheme")
	c.Flags().StringVar(&workspace, "workspace", "", "Path to .xcworkspace")
	c.Flags().StringVar(&project, "project", "", "Path to .xcodeproj")
	c.Flags().StringVar(&product, "product", "", "SwiftPM executable product")
	return c
}

func macosRunCommand() *cobra.Command {
	var mode string
	c := &cobra.Command{
		Use:   "run",
		Short: "Run the project-local macOS build_and_run.sh entrypoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			script := filepath.Join("script", "build_and_run.sh")
			if _, err := os.Stat(script); err != nil {
				return fmt.Errorf("%s not found; run appledev macos bootstrap first", script)
			}
			scriptArgs := []string{}
			switch mode {
			case "run", "":
			case "debug", "logs", "telemetry", "verify":
				scriptArgs = append(scriptArgs, "--"+mode)
			default:
				return fmt.Errorf("unsupported --mode %q (run, debug, logs, telemetry, verify)", mode)
			}
			return runStreaming(cmd, "bash", append([]string{script}, scriptArgs...)...)
		},
	}
	c.Flags().StringVar(&mode, "mode", "run", "run, debug, logs, telemetry, or verify")
	return c
}

func macosLogsCommand() *cobra.Command {
	var process string
	c := &cobra.Command{
		Use:   "logs",
		Short: "Stream macOS unified logs for a process",
		RunE: func(cmd *cobra.Command, args []string) error {
			if process == "" {
				return errors.New("--process is required")
			}
			predicate := fmt.Sprintf(`process == "%s"`, process)
			return runStreaming(cmd, "log", "stream", "--style", "compact", "--predicate", predicate)
		},
	}
	c.Flags().StringVar(&process, "process", "", "macOS process name")
	return c
}

func macosCodesignInspectCommand() *cobra.Command {
	var path string
	c := &cobra.Command{
		Use:   "codesign-inspect",
		Short: "Inspect signing, entitlements, and Gatekeeper assessment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				return errors.New("--path is required")
			}
			if err := runStreaming(cmd, "codesign", "-dv", "--verbose=4", path); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "codesign details failed: %v\n", err)
			}
			if err := runStreaming(cmd, "codesign", "-d", "--entitlements", ":-", path); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "entitlements inspection failed: %v\n", err)
			}
			if err := runStreaming(cmd, "spctl", "--assess", "--type", "execute", "--verbose", path); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Gatekeeper assessment failed: %v\n", err)
			}
			return nil
		},
	}
	c.Flags().StringVar(&path, "path", "", "Path to .app or executable")
	return c
}

func printChecks(cmd *cobra.Command, checks []commandCheck) error {
	var missingRequired []string
	for _, check := range checks {
		_, err := exec.LookPath(check.Name)
		status := "ok"
		if err != nil {
			status = "missing"
			if check.Required {
				missingRequired = append(missingRequired, check.Name)
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-12s %-8s %s\n", check.Name, status, check.Purpose)
	}
	if len(missingRequired) > 0 {
		return fmt.Errorf("missing required command(s): %s", strings.Join(missingRequired, ", "))
	}
	return nil
}

func runStreaming(cmd *cobra.Command, name string, args ...string) error {
	c := exec.CommandContext(cmd.Context(), name, args...)
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()
	c.Stdin = cmd.InOrStdin()
	return c.Run()
}

func inferAppName(scheme, product string) string {
	if scheme != "" {
		return scheme
	}
	return product
}

func macBuildScript(appName, scheme, workspace, project, product string) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, `#!/usr/bin/env bash
set -euo pipefail

APP_NAME=%q
SCHEME=%q
WORKSPACE=%q
PROJECT=%q
PRODUCT=%q
MODE="run"
if [[ $# -gt 0 ]]; then
  case "$1" in
    --debug) MODE="debug" ;;
    --logs) MODE="logs" ;;
    --telemetry) MODE="telemetry" ;;
    --verify) MODE="verify" ;;
    *) echo "unknown mode: $1" >&2; exit 2 ;;
  esac
fi

if pgrep -x "$APP_NAME" >/dev/null 2>&1; then
  pkill -x "$APP_NAME" || true
  sleep 0.5
fi

if [[ -n "$WORKSPACE" || -n "$PROJECT" ]]; then
  args=()
  if [[ -n "$WORKSPACE" ]]; then args+=( -workspace "$WORKSPACE" ); else args+=( -project "$PROJECT" ); fi
  args+=( -scheme "$SCHEME" -configuration Debug -destination 'platform=macOS' build )
  xcodebuild "${args[@]}"
  echo "Built Xcode macOS target. Launch the product from Xcode DerivedData or set up a project-specific bundle path if needed."
elif [[ -n "$PRODUCT" ]]; then
  swift build --product "$PRODUCT"
  BIN="$(swift build --show-bin-path)/$PRODUCT"
  if [[ ! -x "$BIN" ]]; then echo "missing built product: $BIN" >&2; exit 1; fi
  DIST="dist/$APP_NAME.app"
  mkdir -p "$DIST/Contents/MacOS"
  cp "$BIN" "$DIST/Contents/MacOS/$APP_NAME"
  cat > "$DIST/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key><string>$APP_NAME</string>
  <key>CFBundleIdentifier</key><string>local.$APP_NAME</string>
  <key>CFBundleName</key><string>$APP_NAME</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
  <key>NSPrincipalClass</key><string>NSApplication</string>
</dict>
</plist>
PLIST
  APP_BUNDLE="$DIST"
else
  echo "No Xcode scheme/workspace/project or SwiftPM product configured. Re-run appledev macos bootstrap with --scheme or --product." >&2
  exit 2
fi

case "$MODE" in
  run)
    if [[ -n "${APP_BUNDLE:-}" ]]; then /usr/bin/open -n "$APP_BUNDLE"; fi
    ;;
  verify)
    if [[ -n "${APP_BUNDLE:-}" ]]; then /usr/bin/open -n "$APP_BUNDLE"; fi
    sleep 1
    pgrep -x "$APP_NAME" >/dev/null && echo "verified: $APP_NAME is running"
    ;;
  logs)
    if [[ -n "${APP_BUNDLE:-}" ]]; then /usr/bin/open -n "$APP_BUNDLE"; fi
    log stream --style compact --predicate "process == '$APP_NAME'"
    ;;
  telemetry)
    if [[ -n "${APP_BUNDLE:-}" ]]; then /usr/bin/open -n "$APP_BUNDLE"; fi
    log stream --style compact --predicate "process == '$APP_NAME' OR subsystem CONTAINS[c] '$APP_NAME'"
    ;;
  debug)
    if [[ -n "${APP_BUNDLE:-}" ]]; then lldb -- "$APP_BUNDLE/Contents/MacOS/$APP_NAME"; else echo "debug mode needs a concrete executable path; customize this script for the Xcode built product." >&2; exit 2; fi
    ;;
esac
`, appName, scheme, workspace, project, product)
	return b.String()
}
