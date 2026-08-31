package cmd

import (
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/telemetry"
)

func TestRunNotifySlackSemanticValidationIsConciseUsageError(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantError     string
		wantParameter string
	}{
		{
			name:          "invalid webhook",
			args:          []string{"notify", "slack", "--webhook", "https://example.com/services/test", "--message", "hello"},
			wantError:     "Error: --webhook must target hooks.slack.com or hooks.slack-gov.com\n",
			wantParameter: "--webhook",
		},
		{
			name:          "invalid blocks JSON",
			args:          []string{"notify", "slack", "--message", "hello", "--blocks-json", "null"},
			wantError:     "Error: --blocks-json must contain a JSON array\n",
			wantParameter: "--blocks-json",
		},
		{
			name:          "invalid payload JSON",
			args:          []string{"notify", "slack", "--message", "hello", "--payload-json", "null"},
			wantError:     "Error: --payload-json must contain a JSON object\n",
			wantParameter: "--payload-json",
		},
		{
			name:          "attachment option without payload",
			args:          []string{"notify", "slack", "--message", "hello", "--pretext", "release"},
			wantError:     "Error: --pretext and --success require --payload-json or --payload-file\n",
			wantParameter: "--pretext",
		},
		{
			name:          "success option without payload",
			args:          []string{"notify", "slack", "--message", "hello", "--success=false"},
			wantError:     "Error: --pretext and --success require --payload-json or --payload-file\n",
			wantParameter: "--success",
		},
		{
			name:          "invalid thread timestamp",
			args:          []string{"notify", "slack", "--message", "hello", "--thread-ts", "invalid"},
			wantError:     "Error: --thread-ts must be in Slack ts format (e.g. 1733977745.12345)\n",
			wantParameter: "--thread-ts",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetReportFlags(t)
			t.Setenv("ASC_SLACK_WEBHOOK", "https://hooks.slack.com/services/test")
			t.Setenv("ASC_SLACK_WEBHOOK_ALLOW_LOCALHOST", "")

			originalEmitTelemetry := emitTelemetry
			t.Cleanup(func() { emitTelemetry = originalEmitTelemetry })

			var gotExitCode int
			var gotContext telemetry.EventContext
			emitTelemetry = func(_ string, _ string, _ time.Duration, exitCode int, eventContext telemetry.EventContext) {
				gotExitCode = exitCode
				gotContext = eventContext
			}

			stdout, stderr := captureCommandOutput(t, func() {
				if code := Run(test.args, "4.10.0"); code != ExitUsage {
					t.Fatalf("Run() exit code = %d, want %d", code, ExitUsage)
				}
			})

			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if stderr != test.wantError {
				t.Fatalf("stderr = %q, want %q", stderr, test.wantError)
			}
			if gotExitCode != ExitUsage {
				t.Fatalf("telemetry exit code = %d, want %d", gotExitCode, ExitUsage)
			}
			if gotContext.ErrorKind != telemetry.ErrorKindInvalidValue ||
				gotContext.FailureStage != telemetry.FailureStageValidation ||
				gotContext.OutcomeKind != telemetry.OutcomeUsageError ||
				gotContext.FailureParameter != test.wantParameter {
				t.Fatalf("unexpected telemetry context: %+v", gotContext)
			}
		})
	}
}
