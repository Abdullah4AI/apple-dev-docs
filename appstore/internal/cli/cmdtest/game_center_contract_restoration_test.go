package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/gamecenter"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestGameCenterLeaderboardSetMemberLocalizationsViewCompositeOutput(t *testing.T) {
	requestCount := 0
	setGameCenterContractClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.URL.Path != "/v1/gameCenterLeaderboardSetMemberLocalizations/loc-1/gameCenterLeaderboard" {
				t.Fatalf("request 1 path = %q", req.URL.Path)
			}
			return gameCenterContractJSONResponse(`{"data":{"type":"gameCenterLeaderboards","id":"lb-1","attributes":{}}}`), nil
		case 2:
			if req.URL.Path != "/v1/gameCenterLeaderboardSetMemberLocalizations/loc-1/gameCenterLeaderboardSet" {
				t.Fatalf("request 2 path = %q", req.URL.Path)
			}
			return gameCenterContractJSONResponse(`{"data":{"type":"gameCenterLeaderboardSets","id":"set-1","attributes":{}}}`), nil
		case 3:
			if req.URL.Path != "/v1/gameCenterLeaderboardSetMemberLocalizations" {
				t.Fatalf("request 3 path = %q", req.URL.Path)
			}
			query := req.URL.Query()
			if query.Get("filter[gameCenterLeaderboard]") != "lb-1" || query.Get("filter[gameCenterLeaderboardSet]") != "set-1" {
				t.Fatalf("request 3 filters = %q", req.URL.RawQuery)
			}
			return gameCenterContractJSONResponse(`{"data":[{"type":"gameCenterLeaderboardSetMemberLocalizations","id":"loc-1","attributes":{"name":"Top Score","locale":"en-US"}}],"links":{}}`), nil
		default:
			t.Fatalf("unexpected request %d: %s", requestCount, req.URL.String())
			return nil, nil
		}
	}))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"game-center", "leaderboard-sets", "member-localizations", "view", "--id", "loc-1", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if runErr != nil {
		t.Fatalf("run error: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want 3", requestCount)
	}
	var output struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name   string `json:"name"`
				Locale string `json:"locale"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("invalid JSON output %q: %v", stdout, err)
	}
	if output.Data.ID != "loc-1" || output.Data.Attributes.Name != "Top Score" || output.Data.Attributes.Locale != "en-US" {
		t.Fatalf("unexpected output: %+v", output.Data)
	}
}

func TestGameCenterLeaderboardSetMemberLocalizationsViewNotFoundExitCode(t *testing.T) {
	requestCount := 0
	setGameCenterContractClient(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			return gameCenterContractJSONResponse(`{"data":{"type":"gameCenterLeaderboards","id":"lb-1","attributes":{}}}`), nil
		case 2:
			return gameCenterContractJSONResponse(`{"data":{"type":"gameCenterLeaderboardSets","id":"set-1","attributes":{}}}`), nil
		case 3:
			return gameCenterContractJSONResponse(`{"data":[],"links":{}}`), nil
		default:
			t.Fatalf("unexpected request %d: %s", requestCount, req.URL.String())
			return nil, nil
		}
	}))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"game-center", "leaderboard-sets", "member-localizations", "view", "--id", "loc-missing", "--output", "json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, asc.ErrNotFound) {
		t.Fatalf("run error = %v, want ErrNotFound", runErr)
	}
	if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitNotFound {
		t.Fatalf("exit code = %d, want %d", got, rootcmd.ExitNotFound)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want 3", requestCount)
	}
}

func TestGameCenterGroupChallengesSetDeprecatedStubRejectsBeforeHTTP(t *testing.T) {
	tests := []struct {
		name      string
		extraArgs []string
		wants     []string
	}{
		{
			name:  "unsupported operation",
			wants: []string{"deprecated", "does not support replacing", "asc game-center challenges create --group-id"},
		},
		{name: "output", extraArgs: []string{"--output", "json"}, wants: []string{"deprecated", "omit --output and --pretty"}},
		{name: "pretty", extraArgs: []string{"--pretty=false"}, wants: []string{"deprecated", "omit --output and --pretty"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factoryCalled := false
			restore := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				factoryCalled = true
				return nil, errors.New("poison client factory called")
			})
			t.Cleanup(restore)

			args := append([]string{"game-center", "groups", "challenges", "set", "--group-id", "group-1", "--ids", "challenge-1"}, test.extraArgs...)
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				runErr = root.Run(context.Background())
			})

			if runErr == nil || !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("run error = %v, want usage error", runErr)
			}
			if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d", got, rootcmd.ExitUsage)
			}
			if factoryCalled {
				t.Fatal("client factory called for deprecated challenge setter")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			for _, want := range test.wants {
				if !strings.Contains(strings.ToLower(stderr), strings.ToLower(want)) {
					t.Fatalf("stderr = %q, want %q", stderr, want)
				}
			}
			if len(test.extraArgs) > 0 && strings.Count(stderr, "omit --output and --pretty") != 1 {
				t.Fatalf("stderr = %q, want output guidance exactly once", stderr)
			}
		})
	}
}

func TestGameCenterDetailsListDeprecatedPaginationFlagsRejectBeforeHTTP(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
	}{
		{name: "limit", args: []string{"--limit", "0"}, flag: "--limit"},
		{name: "next", args: []string{"--next", ""}, flag: "--next"},
		{name: "paginate", args: []string{"--paginate=false"}, flag: "--paginate"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factoryCalled := false
			restore := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				factoryCalled = true
				return nil, errors.New("poison client factory called")
			})
			t.Cleanup(restore)

			args := append([]string{"game-center", "details", "list", "--app", "app-1"}, test.args...)
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				runErr = root.Run(context.Background())
			})

			if runErr == nil || !errors.Is(runErr, flag.ErrHelp) {
				t.Fatalf("run error = %v, want usage error", runErr)
			}
			if got := rootcmd.ExitCodeFromError(runErr); got != rootcmd.ExitUsage {
				t.Fatalf("exit code = %d, want %d", got, rootcmd.ExitUsage)
			}
			if factoryCalled {
				t.Fatalf("client factory called for deprecated %s", test.flag)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, test.flag+" is deprecated") || !strings.Contains(stderr, "single Game Center detail") {
				t.Fatalf("stderr = %q, want precise %s singleton guidance", stderr, test.flag)
			}
			if strings.Contains(stderr, "flag provided but not defined") {
				t.Fatalf("stderr contains generic unknown-flag failure: %q", stderr)
			}
		})
	}
}

func TestGameCenterRestoredContractsAppearInHelp(t *testing.T) {
	detailsHelp := shared.DefaultUsageFunc(gamecenter.GameCenterDetailsListCommand())
	for _, want := range []string{"--limit", "--next", "--paginate", "Deprecated"} {
		if !strings.Contains(detailsHelp, want) {
			t.Fatalf("details help missing %q:\n%s", want, detailsHelp)
		}
	}

	setHelp := shared.DefaultUsageFunc(gamecenter.GameCenterGroupChallengesSetCommand())
	for _, want := range []string{"DEPRECATED", "--group-id", "--ids", "--output", "--pretty", "does not support"} {
		if !strings.Contains(setHelp, want) {
			t.Fatalf("challenge set help missing %q:\n%s", want, setHelp)
		}
	}

	memberHelp := shared.DefaultUsageFunc(gamecenter.GameCenterLeaderboardSetMemberLocalizationsCommand())
	if !strings.Contains(memberHelp, "view") {
		t.Fatalf("member-localizations help missing restored view command:\n%s", memberHelp)
	}
}

func setGameCenterContractClient(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	setupAuth(t)
	client, err := asc.NewClientWithHTTPClient(
		os.Getenv("ASC_KEY_ID"),
		os.Getenv("ASC_ISSUER_ID"),
		os.Getenv("ASC_PRIVATE_KEY_PATH"),
		&http.Client{Transport: transport},
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	restore := shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return client, nil
	})
	t.Cleanup(restore)
}

func gameCenterContractJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
