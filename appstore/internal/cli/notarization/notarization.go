package notarization

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

// NotarizationCommand returns the notarization command group.
func NotarizationCommand() *ffcli.Command {
	return notarizationCommand()
}

// notarizationCommand returns the top-level notarization command.
func notarizationCommand() *ffcli.Command {
	return &ffcli.Command{
		Name:       "notarization",
		ShortUsage: "asc notarization <subcommand> [flags]",
		ShortHelp:  "Manage macOS notarization submissions.",
		LongHelp: `Manage macOS notarization submissions via the Apple Notary API.

Examples:
  asc notarization submit --file ./MyApp.zip
  asc notarization submit --file ./MyApp.zip --wait
  asc notarization status --id "SUBMISSION_ID"
  asc notarization log --id "SUBMISSION_ID"
  asc notarization list`,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			submitCommand(),
			statusCommand(),
			logCommand(),
			listCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// submitCommand returns the submit subcommand.
func submitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notarization submit", flag.ExitOnError)

	filePath := fs.String("file", "", "Path to the file to notarize (required, zip/dmg/pkg)")
	wait := fs.Bool("wait", false, "Wait for notarization to complete")
	pollInterval := fs.String("poll-interval", "15s", "Polling interval when using --wait")
	timeout := fs.String("timeout", "30m", "Timeout when using --wait")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "submit",
		ShortUsage: "asc notarization submit --file <path> [flags]",
		ShortHelp:  "Submit software for notarization.",
		LongHelp: `Submit a file for macOS notarization via the Apple Notary API.

The file must be a zip, dmg, or pkg archive. The command computes the file's
SHA-256 hash, creates a submission, uploads the file to Apple's S3 bucket,
and optionally waits for the notarization to complete.

Examples:
  asc notarization submit --file ./MyApp.zip
  asc notarization submit --file ./MyApp.zip --wait
  asc notarization submit --file ./MyApp.zip --wait --poll-interval 30s --timeout 1h
  asc notarization submit --file ./MyApp.zip --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			pathValue := strings.TrimSpace(*filePath)
			if pathValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --file is required")
				return shared.MissingRequiredUsageError()
			}

			interval, err := time.ParseDuration(strings.TrimSpace(*pollInterval))
			if err != nil || interval <= 0 {
				return fmt.Errorf("notarization submit: --poll-interval must be a valid positive duration (e.g. 15s, 1m)")
			}

			timeoutDuration, err := time.ParseDuration(strings.TrimSpace(*timeout))
			if err != nil || timeoutDuration <= 0 {
				return fmt.Errorf("notarization submit: --timeout must be a valid positive duration (e.g. 30m, 1h)")
			}

			// Preserve the explicit symlink error while relying on the no-follow
			// open below for the actual security boundary.
			pathInfo, err := os.Lstat(pathValue)
			if err != nil {
				return fmt.Errorf("notarization submit: %w", err)
			}
			if pathInfo.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("notarization submit: refusing to read symlink %q", pathValue)
			}
			if pathInfo.IsDir() {
				return fmt.Errorf("notarization submit: %q is a directory", pathValue)
			}
			if !pathInfo.Mode().IsRegular() {
				return fmt.Errorf("notarization submit: %q is not a regular file", pathValue)
			}

			fileHandle, err := secureopen.OpenExistingNoFollow(pathValue)
			if err != nil {
				return fmt.Errorf("notarization submit: failed to open file: %w", err)
			}
			defer fileHandle.Close()

			info, err := fileHandle.Stat()
			if err != nil {
				return fmt.Errorf("notarization submit: failed to stat opened file: %w", err)
			}
			if info.IsDir() {
				return fmt.Errorf("notarization submit: %q is a directory", pathValue)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("notarization submit: %q is not a regular file", pathValue)
			}
			if info.Size() <= 0 {
				return fmt.Errorf("notarization submit: file must not be empty")
			}

			ext := strings.ToLower(filepath.Ext(pathValue))
			if ext != ".zip" && ext != ".dmg" && ext != ".pkg" {
				return fmt.Errorf("notarization submit: unsupported file type %q (must be .zip, .dmg, or .pkg)", ext)
			}

			// Compute SHA-256
			if shared.ProgressEnabled() {
				fmt.Fprintf(os.Stderr, "Computing SHA-256 hash of %s...\n", pathValue)
			}
			sha256Hash, err := asc.ComputeFileSHA256(fileHandle)
			if err != nil {
				return fmt.Errorf("notarization submit: failed to compute SHA-256: %w", err)
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("notarization submit: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			// Submit to Notary API
			submissionName := info.Name()
			if shared.ProgressEnabled() {
				fmt.Fprintf(os.Stderr, "Submitting %s for notarization...\n", submissionName)
			}

			submitResp, err := client.SubmitNotarization(requestCtx, sha256Hash, submissionName)
			if err != nil {
				return fmt.Errorf("notarization submit: %w", err)
			}

			submissionID := submitResp.Data.ID
			if shared.ProgressEnabled() {
				fmt.Fprintf(os.Stderr, "Submission created: %s\n", submissionID)
			}

			// Upload file to S3
			if shared.ProgressEnabled() {
				fmt.Fprintf(os.Stderr, "Uploading %s to Apple...\n", submissionName)
			}

			uploadCtx, uploadCancel := shared.ContextWithUploadTimeout(ctx)
			defer uploadCancel()

			creds := asc.S3Credentials{
				AccessKeyID:     submitResp.Data.Attributes.AwsAccessKeyID,
				SecretAccessKey: submitResp.Data.Attributes.AwsSecretAccessKey,
				SessionToken:    submitResp.Data.Attributes.AwsSessionToken,
				Bucket:          submitResp.Data.Attributes.Bucket,
				Object:          submitResp.Data.Attributes.Object,
			}

			contentType := notaryContentType(pathValue)
			if err := asc.UploadToS3(uploadCtx, creds, fileHandle, sha256Hash, info.Size(), contentType); err != nil {
				return fmt.Errorf("notarization submit: upload failed: %w", err)
			}

			if shared.ProgressEnabled() {
				fmt.Fprintln(os.Stderr, "Upload complete.")
			}

			// If not waiting, print the submission response and exit
			if !*wait {
				if shared.ProgressEnabled() {
					fmt.Fprintf(os.Stderr, "Use 'asc notarization status --id %s' to check progress.\n", submissionID)
				}
				resp := &asc.NotarySubmissionStatusResponse{
					Data: asc.NotarySubmissionStatusData{
						ID:   submissionID,
						Type: "submissions",
						Attributes: asc.NotarySubmissionStatusAttributes{
							Status:      asc.NotaryStatusInProgress,
							Name:        submissionName,
							CreatedDate: "",
						},
					},
				}
				return shared.PrintOutput(resp, *output.Output, *output.Pretty)
			}

			// Wait for notarization to complete
			if shared.ProgressEnabled() {
				fmt.Fprintf(os.Stderr, "Waiting for notarization (polling every %s, timeout %s)...\n", interval, timeoutDuration)
			}

			waitCtx, waitCancel := context.WithTimeout(ctx, timeoutDuration)
			defer waitCancel()

			statusResp, err := waitForNotarization(waitCtx, client, submissionID, interval)
			if err != nil {
				return fmt.Errorf("notarization submit: %w", err)
			}

			if err := shared.PrintOutput(statusResp, *output.Output, *output.Pretty); err != nil {
				return err
			}

			switch statusResp.Data.Attributes.Status {
			case asc.NotaryStatusAccepted:
				if shared.ProgressEnabled() {
					fmt.Fprintln(os.Stderr, "Notarization complete! Status: Accepted")
				}
				return nil
			case asc.NotaryStatusInvalid, asc.NotaryStatusRejected:
				if shared.ProgressEnabled() {
					fmt.Fprintf(os.Stderr, "Notarization failed. Status: %s\n", statusResp.Data.Attributes.Status)
					fmt.Fprintf(os.Stderr, "Run 'asc notarization log --id %s' for details.\n", submissionID)
				}
				return shared.NewReportedError(fmt.Errorf("notarization %s: %s", submissionID, statusResp.Data.Attributes.Status))
			default:
				return nil
			}
		},
	}
}

// statusCommand returns the status subcommand.
func statusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notarization status", flag.ExitOnError)

	submissionID := fs.String("id", "", "Submission ID (required)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "asc notarization status --id \"SUBMISSION_ID\"",
		ShortHelp:  "Get the status of a notarization submission.",
		LongHelp: `Get the status of a notarization submission.

Status values: Accepted, In Progress, Invalid, Rejected.

Examples:
  asc notarization status --id "SUBMISSION_ID"
  asc notarization status --id "SUBMISSION_ID" --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			idValue := strings.TrimSpace(*submissionID)
			if idValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return shared.MissingRequiredUsageError()
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("notarization status: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			resp, err := client.GetNotarizationStatus(requestCtx, idValue)
			if err != nil {
				return fmt.Errorf("notarization status: failed to fetch: %w", err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

// logCommand returns the log subcommand.
func logCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notarization log", flag.ExitOnError)

	submissionID := fs.String("id", "", "Submission ID (required)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "log",
		ShortUsage: "asc notarization log --id \"SUBMISSION_ID\"",
		ShortHelp:  "Get the developer log URL for a notarization submission.",
		LongHelp: `Get the developer log URL for a notarization submission.

The log contains detailed information about the notarization result,
including any issues found during the scan.

Examples:
  asc notarization log --id "SUBMISSION_ID"
  asc notarization log --id "SUBMISSION_ID" --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			idValue := strings.TrimSpace(*submissionID)
			if idValue == "" {
				fmt.Fprintln(os.Stderr, "Error: --id is required")
				return shared.MissingRequiredUsageError()
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("notarization log: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			resp, err := client.GetNotarizationLogs(requestCtx, idValue)
			if err != nil {
				return fmt.Errorf("notarization log: failed to fetch: %w", err)
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

// listCommand returns the list subcommand.
func listCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notarization list", flag.ExitOnError)

	limit := fs.Int("limit", 0, "Maximum number of results to display (0 = all)")
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "list",
		ShortUsage: "asc notarization list [flags]",
		ShortHelp:  "List previous notarization submissions.",
		LongHelp: `List previous notarization submissions.

Examples:
  asc notarization list
  asc notarization list --limit 5
  asc notarization list --output table`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *limit < 0 {
				return fmt.Errorf("notarization list: --limit must not be negative")
			}

			client, err := shared.GetASCClient()
			if err != nil {
				return fmt.Errorf("notarization list: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			resp, err := client.ListNotarizations(requestCtx)
			if err != nil {
				return fmt.Errorf("notarization list: failed to fetch: %w", err)
			}

			if *limit > 0 && len(resp.Data) > *limit {
				resp.Data = resp.Data[:*limit]
			}

			return shared.PrintOutput(resp, *output.Output, *output.Pretty)
		},
	}
}

// notarizationPollMaxBackoff caps the delay applied after repeated transient
// status-check failures.
const notarizationPollMaxBackoff = 2 * time.Minute

// waitForNotarization polls the notarization status until the submission reaches
// a terminal state, the wait deadline expires, or a status check fails for a
// reason that will not resolve on its own. Transient failures (transport errors,
// request timeouts, and retryable HTTP statuses) are reported on stderr and
// retried with backoff, because the archive is already uploaded and the
// submission keeps progressing server-side.
func waitForNotarization(ctx context.Context, client *asc.Client, submissionID string, pollInterval time.Duration) (*asc.NotarySubmissionStatusResponse, error) {
	consecutiveFailures := 0
	var lastTransientErr error

	for {
		requestCtx, cancel := shared.ContextWithTimeout(ctx)
		resp, err := client.GetNotarizationStatus(requestCtx, submissionID)
		cancel()

		delay := pollInterval
		switch {
		case err == nil:
			consecutiveFailures = 0
			lastTransientErr = nil

			switch resp.Data.Attributes.Status {
			case asc.NotaryStatusAccepted, asc.NotaryStatusInvalid, asc.NotaryStatusRejected:
				return resp, nil
			default:
				// Treat unknown statuses (including InProgress) as non-terminal and continue polling
				if shared.ProgressEnabled() {
					fmt.Fprintf(os.Stderr, "Status: %s (checking again in %s)\n", resp.Data.Attributes.Status, pollInterval)
				}
			}
		case ctx.Err() != nil:
			// The wait deadline expired (or the caller cancelled) while the
			// status request was still in flight.
			return nil, notarizationWaitEndedError(ctx, lastTransientErr)
		case isTransientNotarizationPollError(err):
			consecutiveFailures++
			lastTransientErr = err
			delay = notarizationPollBackoff(pollInterval, consecutiveFailures)
			fmt.Fprintf(os.Stderr, "Warning: notarization status check failed (%v); retrying in %s\n", err, delay)
		default:
			return nil, fmt.Errorf("failed to check status: %w", err)
		}

		if !waitBeforeNextNotarizationPoll(ctx, delay) {
			return nil, notarizationWaitEndedError(ctx, lastTransientErr)
		}
	}
}

// waitBeforeNextNotarizationPoll sleeps for delay and reports whether the wait
// context is still live.
func waitBeforeNextNotarizationPoll(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// notarizationWaitEndedError explains why the wait stopped once the wait context
// finished, preserving the most recent transient status-check failure.
func notarizationWaitEndedError(ctx context.Context, lastTransientErr error) error {
	reason := "timed out waiting for notarization"
	if errors.Is(ctx.Err(), context.Canceled) {
		reason = "canceled while waiting for notarization"
	}
	if lastTransientErr != nil {
		return fmt.Errorf("%s (last status check failed: %w): %w", reason, lastTransientErr, ctx.Err())
	}
	return fmt.Errorf("%s: %w", reason, ctx.Err())
}

// isTransientNotarizationPollError reports whether a status-check failure is
// worth retrying. The Notary API path has no client-side retry wrapper, so the
// wait loop classifies transport failures, per-request timeouts, and retryable
// HTTP statuses itself.
func isTransientNotarizationPollError(err error) bool {
	if err == nil {
		return false
	}
	if asc.IsRetryable(err) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// Only the per-request timeout reaches here; callers check the wait
		// context before classifying.
		return true
	}

	var statusErr interface{ HTTPStatusCode() int }
	if errors.As(err, &statusErr) {
		switch statusErr.HTTPStatusCode() {
		case http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed)
}

// notarizationPollBackoff spaces out retries after consecutive transient
// failures without ever polling faster than the caller's interval.
func notarizationPollBackoff(pollInterval time.Duration, consecutiveFailures int) time.Duration {
	maxBackoff := notarizationPollMaxBackoff
	if pollInterval > maxBackoff {
		maxBackoff = pollInterval
	}

	delay := pollInterval
	for i := 1; i < consecutiveFailures; i++ {
		if delay >= maxBackoff {
			return maxBackoff
		}
		delay *= 2
	}
	if delay > maxBackoff {
		return maxBackoff
	}
	return delay
}

func notaryContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".zip":
		return "application/zip"
	case ".dmg":
		return "application/x-apple-diskimage"
	case ".pkg":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}
