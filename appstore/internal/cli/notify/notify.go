package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const (
	slackWebhookEnvVar               = "APPSTORE_SLACK_WEBHOOK"
	slackWebhookAllowLocalEnv        = "APPSTORE_SLACK_WEBHOOK_ALLOW_LOCALHOST"
	slackWebhookHost                 = "hooks.slack.com"
	slackWebhookPathPrefix           = "/services/"
	slackWebhookMaxResponseBodyBytes = 4096
)

var slackHTTPClient = func() *http.Client {
	return &http.Client{Timeout: asc.ResolveTimeout()}
}

func slackFlags(fs *flag.FlagSet) (webhook *string, channel *string, message *string, blocksJSON *string, blocksFile *string) {
	webhook = fs.String("webhook", "", "Slack webhook URL (or set "+slackWebhookEnvVar+" env var)")
	channel = fs.String("channel", "", "Slack channel (#channel or @username)")
	message = fs.String("message", "", "Message to send to Slack")
	blocksJSON = fs.String("blocks-json", "", "Slack Block Kit JSON array")
	blocksFile = fs.String("blocks-file", "", "Path to Slack Block Kit JSON array file")
	return
}

func NotifyCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notify", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "notify",
		ShortUsage: "appstore notify <subcommand> [flags]",
		ShortHelp:  "Send notifications to external services.",
		LongHelp: `Send notifications to external services.

Examples:
  appstore notify slack --webhook $WEBHOOK --message "Build uploaded"
  APPSTORE_SLACK_WEBHOOK=$WEBHOOK appstore notify slack --message "Done"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			SlackCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

func SlackCommand() *ffcli.Command {
	fs := flag.NewFlagSet("notify slack", flag.ExitOnError)

	webhook, channel, message, blocksJSON, blocksFile := slackFlags(fs)

	return &ffcli.Command{
		Name:       "slack",
		ShortUsage: "appstore notify slack --webhook URL --message TEXT",
		ShortHelp:  "Send a message to Slack via webhook.",
		LongHelp: `Send a message to Slack via incoming webhook.

This command sends a JSON payload to a Slack incoming webhook URL.
The webhook URL can be provided via --webhook flag or APPSTORE_SLACK_WEBHOOK env var.
When using blocks, keep --message as the top-level text fallback.

Examples:
  appstore notify slack --webhook "https://hooks.slack.com/..." --message "Build uploaded"
  appstore notify slack --message "Done" --channel "#deployments"
  APPSTORE_SLACK_WEBHOOK=$WEBHOOK appstore notify slack --message "Release v2.1 ready"
  appstore notify slack --message "Release ready" --blocks-json '[{"type":"section","text":{"type":"mrkdwn","text":"*Release* ready"}}]'
  appstore notify slack --message "Release ready" --blocks-file ./blocks.json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			webhookURL := resolveWebhook(*webhook)
			if webhookURL == "" {
				fmt.Fprintf(os.Stderr, "Error: --webhook is required or set %s env var\n", slackWebhookEnvVar)
				return flag.ErrHelp
			}
			if err := validateSlackWebhookURL(webhookURL); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err.Error())
				return flag.ErrHelp
			}

			msg := strings.TrimSpace(*message)
			if msg == "" {
				fmt.Fprintln(os.Stderr, "Error: --message is required")
				return flag.ErrHelp
			}

			blocks, err := parseSlackBlocks(*blocksJSON, *blocksFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err.Error())
				return flag.ErrHelp
			}

			payload := map[string]any{}
			payload["text"] = msg

			if ch := strings.TrimSpace(*channel); ch != "" {
				payload["channel"] = ch
			}
			if blocks != nil {
				payload["blocks"] = blocks
			}

			body, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("notify slack: failed to marshal payload: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			req, err := http.NewRequestWithContext(requestCtx, "POST", webhookURL, bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("notify slack: failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := slackHTTPClient()
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("notify slack: failed to send: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				limited := io.LimitReader(resp.Body, slackWebhookMaxResponseBodyBytes)
				respBody, readErr := io.ReadAll(limited)
				if readErr != nil {
					return fmt.Errorf("notify slack: failed to read response: %w", readErr)
				}
				message := strings.TrimSpace(string(respBody))
				if message == "" {
					return fmt.Errorf("notify slack: unexpected response %d", resp.StatusCode)
				}
				return fmt.Errorf("notify slack: unexpected response %d: %s", resp.StatusCode, message)
			}

			fmt.Fprintln(os.Stderr, "Message sent to Slack successfully")
			return nil
		},
	}
}

func resolveWebhook(flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(slackWebhookEnvVar)); v != "" {
		return v
	}
	return ""
}

func parseSlackBlocks(blocksJSON string, blocksFile string) ([]json.RawMessage, error) {
	blocksJSON = strings.TrimSpace(blocksJSON)
	blocksFile = strings.TrimSpace(blocksFile)

	if blocksJSON != "" && blocksFile != "" {
		return nil, fmt.Errorf("only one of --blocks-json or --blocks-file may be set")
	}
	if blocksJSON == "" && blocksFile == "" {
		return nil, nil
	}

	source := "--blocks-json"
	if blocksFile != "" {
		data, err := os.ReadFile(blocksFile)
		if err != nil {
			return nil, fmt.Errorf("--blocks-file must be readable: %w", err)
		}
		blocksJSON = strings.TrimSpace(string(data))
		source = "--blocks-file"
	}
	if blocksJSON == "" {
		return nil, fmt.Errorf("%s must contain a JSON array", source)
	}

	var blocks []json.RawMessage
	if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
		return nil, fmt.Errorf("%s must contain a JSON array: %w", source, err)
	}

	return blocks, nil
}

func validateSlackWebhookURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("--webhook must be a valid Slack webhook URL (https://hooks.slack.com/...)")
	}
	host := strings.ToLower(parsed.Hostname())
	if allowLocalSlackWebhook() && isLocalhost(host) {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("--webhook must use http or https")
		}
		return nil
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("--webhook must use https")
	}
	if ip := net.ParseIP(host); ip != nil {
		return fmt.Errorf("--webhook must target hooks.slack.com")
	}
	if host != slackWebhookHost {
		return fmt.Errorf("--webhook must target hooks.slack.com")
	}
	if !strings.HasPrefix(parsed.Path, slackWebhookPathPrefix) {
		return fmt.Errorf("--webhook must start with %s", slackWebhookPathPrefix)
	}
	return nil
}

func allowLocalSlackWebhook() bool {
	value := strings.TrimSpace(os.Getenv(slackWebhookAllowLocalEnv))
	return value == "1" || strings.EqualFold(value, "true")
}

func isLocalhost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
