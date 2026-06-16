package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func executeCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestIOSCommandExposesPluginFeatures(t *testing.T) {
	out, _, err := executeCommandForTest(newIOSCommand(), "features")
	if err != nil {
		t.Fatalf("features returned error: %v", err)
	}
	for _, want := range []string{"ios-debugger-agent", "ios-simulator-browser", "ios-ettrace-performance", "ios-memgraph-leaks"} {
		if !strings.Contains(out, want) {
			t.Fatalf("features output missing %q\n%s", want, out)
		}
	}
}

func TestIOSMCPConfigMatchesCodexPlugin(t *testing.T) {
	out, _, err := executeCommandForTest(newIOSCommand(), "mcp-config")
	if err != nil {
		t.Fatalf("mcp-config returned error: %v", err)
	}
	for _, want := range []string{"xcodebuildmcp", "npx", "xcodebuildmcp@latest", "simulator,ui-automation,debugging,logging"} {
		if !strings.Contains(out, want) {
			t.Fatalf("mcp config missing %q\n%s", want, out)
		}
	}
}

func TestMacOSCommandExposesPluginFeatures(t *testing.T) {
	out, _, err := executeCommandForTest(newMacOSCommand(), "features")
	if err != nil {
		t.Fatalf("features returned error: %v", err)
	}
	for _, want := range []string{"build-run-debug", "signing-entitlements", "swiftpm-macos", "telemetry"} {
		if !strings.Contains(out, want) {
			t.Fatalf("features output missing %q\n%s", want, out)
		}
	}
}

func TestMacBuildScriptStagesSwiftPMGUIBundle(t *testing.T) {
	script := macBuildScript("SmokeApp", "", "", "", "SmokeApp")
	for _, want := range []string{"swift build --product", "dist/$APP_NAME.app", "/usr/bin/open -n", "NSPrincipalClass", "--verify"} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}
}
