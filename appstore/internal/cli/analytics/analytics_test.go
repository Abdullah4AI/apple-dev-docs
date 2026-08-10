package analytics

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestAnalyticsViewProcessingDateFlagLifecycle(t *testing.T) {
	cmd := AnalyticsGetCommand()
	if cmd.FlagSet.Lookup("processing-date") == nil {
		t.Fatal("canonical --processing-date flag is not registered")
	}
	if cmd.FlagSet.Lookup("date") == nil {
		t.Fatal("deprecated --date compatibility flag is not registered")
	}

	visible := make(map[string]bool)
	for _, item := range shared.VisibleHelpFlags(cmd.FlagSet) {
		visible[item.Name] = true
	}
	if !visible["processing-date"] {
		t.Fatal("canonical --processing-date flag is hidden from help")
	}
	if visible["date"] {
		t.Fatal("deprecated --date flag should be hidden from canonical help")
	}

	usage := cmd.UsageFunc(cmd)
	for _, want := range []string{
		"The --processing-date and --granularity filters are sent to App Store Connect",
		"The deprecated --date compatibility flag retains its previous local matching",
		`--processing-date "2024-01-20"`,
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("analytics view help does not contain %q:\n%s", want, usage)
		}
	}
	if strings.Contains(usage, `--date "2024-01-20"`) {
		t.Fatalf("analytics view help still teaches deprecated --date:\n%s", usage)
	}
}

func TestAnalyticsRequestsAccessTypeFlagLifecycle(t *testing.T) {
	cmd := AnalyticsRequestsCommand()
	if cmd.FlagSet.Lookup("access-type") == nil {
		t.Fatal("canonical --access-type flag is not registered")
	}
	visible := make(map[string]bool)
	for _, item := range shared.VisibleHelpFlags(cmd.FlagSet) {
		visible[item.Name] = true
	}
	if !visible["access-type"] {
		t.Fatal("canonical --access-type flag is hidden from help")
	}
	usage := cmd.UsageFunc(cmd)
	if !strings.Contains(usage, `--access-type ONGOING`) {
		t.Fatalf("analytics requests help does not teach --access-type:\n%s", usage)
	}
	if cmd.FlagSet.Lookup("state") == nil {
		t.Fatal("deprecated --state compatibility flag is not registered")
	}
	for _, item := range shared.VisibleHelpFlags(cmd.FlagSet) {
		if item.Name == "state" {
			t.Fatal("deprecated --state flag should be hidden from canonical help")
		}
	}
}

func TestAnalyticsRequestsRejectsInvalidAccessType(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{
		"analytics", "requests",
		"--app", "app-1",
		"--access-type", "COMPLETED",
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--access-type must be ONGOING or ONE_TIME_SNAPSHOT") {
		t.Fatalf("expected access type validation error, got %q", stderr)
	}
}

func TestAnalyticsRequestsDeprecatedStateFailsBeforeAuth(t *testing.T) {
	tests := []struct {
		name      string
		stateArgs []string
	}{
		{name: "legacy value", stateArgs: []string{"--state", "COMPLETED"}},
		{name: "equals empty", stateArgs: []string{"--state="}},
		{name: "separate empty", stateArgs: []string{"--state", ""}},
		{name: "whitespace", stateArgs: []string{"--state", " \t "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientFactoryCalls := 0
			restoreClient := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				clientFactoryCalls++
				return nil, errors.New("client construction should not be attempted")
			})
			t.Cleanup(restoreClient)

			args := []string{"analytics", "requests", "--app", "app-1"}
			args = append(args, tt.stateArgs...)
			stdout, stderr, err := runAnalyticsCommand(t, args)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("expected usage error, got %v", err)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, "--state is deprecated and unsupported by App Store Connect") ||
				!strings.Contains(stderr, "use --access-type ONGOING or --access-type ONE_TIME_SNAPSHOT") {
				t.Fatalf("expected migration guidance, got %q", stderr)
			}
			if clientFactoryCalls != 0 {
				t.Fatalf("client factory calls = %d, want 0", clientFactoryCalls)
			}
		})
	}
}

func TestAnalyticsSalesRejectsUnsupportedContractBeforeAuth(t *testing.T) {
	t.Setenv("ASC_KEY_ID", "")
	t.Setenv("ASC_ISSUER_ID", "")
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY", "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name: "invalid tuple",
			args: []string{
				"analytics", "sales", "--vendor", "12345678",
				"--type", "WIN_BACK_ELIGIBILITY", "--subtype", "DETAILED", "--frequency", "WEEKLY", "--date", "2026-08-02",
			},
			wantErr: "unsupported sales report combination",
		},
		{
			name: "version outside tuple",
			args: []string{
				"analytics", "sales", "--vendor", "12345678",
				"--type", "SALES", "--subtype", "SUMMARY", "--frequency", "DAILY", "--version", "1_5",
			},
			wantErr: "--version 1_5 is not supported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runAnalyticsCommand(t, test.args)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("expected usage error, got %v", err)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected %q, got %q", test.wantErr, stderr)
			}
		})
	}
}

func parseAnalyticsArgs(args []string) []string {
	if len(args) > 0 && args[0] == "analytics" {
		return args[1:]
	}
	return args
}

func runAnalyticsCommand(t *testing.T, args []string) (string, string, error) {
	t.Helper()

	cmd := AnalyticsCommand()
	cmd.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Parse(parseAnalyticsArgs(args)); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = cmd.Run(context.Background())
	})

	return stdout, stderr, runErr
}

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		_ = rOut.Close()
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		_ = rErr.Close()
		errC <- buf.String()
	}()

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_ = wOut.Close()
		_ = wErr.Close()
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()

	stdout := <-outC
	stderr := <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return stdout, stderr
}

func TestAnalyticsSalesValidationErrors(t *testing.T) {
	t.Setenv("ASC_VENDOR_NUMBER", "")
	t.Setenv("ASC_ANALYTICS_VENDOR_NUMBER", "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing vendor",
			args:    []string{"analytics", "sales", "--type", "SALES", "--subtype", "SUMMARY", "--frequency", "DAILY", "--date", "2024-01-20"},
			wantErr: "--vendor is required",
		},
		{
			name:    "missing type",
			args:    []string{"analytics", "sales", "--vendor", "12345678", "--subtype", "SUMMARY", "--frequency", "DAILY", "--date", "2024-01-20"},
			wantErr: "--type is required",
		},
		{
			name:    "missing subtype",
			args:    []string{"analytics", "sales", "--vendor", "12345678", "--type", "SALES", "--frequency", "DAILY", "--date", "2024-01-20"},
			wantErr: "--subtype is required",
		},
		{
			name:    "missing frequency",
			args:    []string{"analytics", "sales", "--vendor", "12345678", "--type", "SALES", "--subtype", "SUMMARY", "--date", "2024-01-20"},
			wantErr: "--frequency is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runAnalyticsCommand(t, test.args)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", err)
			}

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}

func TestAnalyticsSalesRequiresDateForNonDailyFrequency(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{
		"analytics", "sales",
		"--vendor", "12345678",
		"--type", "SALES",
		"--subtype", "SUMMARY",
		"--frequency", "WEEKLY",
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--date is required for WEEKLY reports") {
		t.Fatalf("expected conditional date error, got %q", stderr)
	}
}

func TestAnalyticsRequestValidationErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing app",
			args:    []string{"analytics", "request", "--access-type", "ONGOING"},
			wantErr: "--app is required",
		},
		{
			name:    "missing access type",
			args:    []string{"analytics", "request", "--app", "APP_ID"},
			wantErr: "--access-type is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runAnalyticsCommand(t, test.args)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", err)
			}

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}

func TestAnalyticsRequestsValidationErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "requests"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--app is required") {
		t.Fatalf("expected missing app error, got %q", stderr)
	}
}

func TestAnalyticsGetValidationErrors(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "view"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--request-id is required") {
		t.Fatalf("expected missing request-id error, got %q", stderr)
	}
}

func TestAnalyticsReportsGetValidationErrors(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "reports", "view"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--report-id is required") {
		t.Fatalf("expected missing report-id error, got %q", stderr)
	}
}

func TestAnalyticsReportsRelationshipsValidationErrors(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "reports", "links"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--report-id is required") {
		t.Fatalf("expected missing report-id error, got %q", stderr)
	}
}

func TestAnalyticsInstancesGetValidationErrors(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "instances", "view"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--instance-id is required") {
		t.Fatalf("expected missing instance-id error, got %q", stderr)
	}
}

func TestAnalyticsInstancesRelationshipsValidationErrors(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "instances", "links"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--instance-id is required") {
		t.Fatalf("expected missing instance-id error, got %q", stderr)
	}
}

func TestAnalyticsSegmentsGetValidationErrors(t *testing.T) {
	stdout, stderr, err := runAnalyticsCommand(t, []string{"analytics", "segments", "view"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--segment-id is required") {
		t.Fatalf("expected missing segment-id error, got %q", stderr)
	}
}

func TestAnalyticsDownloadValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing request id",
			args:    []string{"analytics", "download"},
			wantErr: "--request-id is required",
		},
		{
			name:    "missing instance id",
			args:    []string{"analytics", "download", "--request-id", "11111111-1111-1111-1111-111111111111"},
			wantErr: "--instance-id is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runAnalyticsCommand(t, test.args)
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", err)
			}

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}
