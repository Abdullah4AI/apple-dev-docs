package cmdtest

import (
	"encoding/json"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

type schemaEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func TestSchemaSupportsDocumentedTrailingPrettyFlag(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "builds", "--pretty"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.HasPrefix(stdout, "[\n  {") {
		t.Fatalf("expected pretty-printed JSON, got %q", stdout)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	if len(endpoints) == 0 {
		t.Fatal("expected endpoints matching builds")
	}
}

func TestSchemaSupportsMethodFlagAfterQuery(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "apps", "--method", "POST"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	if len(endpoints) == 0 {
		t.Fatal("expected POST endpoints matching apps")
	}
	for _, endpoint := range endpoints {
		if endpoint.Method != "POST" {
			t.Fatalf("expected only POST endpoints, got %#v", endpoints)
		}
	}
}

func TestSchemaMethodAndPathQueryIsExact(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "GET /v1/apps"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.Contains(stdout, "getAction") {
		t.Fatalf("expected index-only cardinality metadata to remain internal, got %q", stdout)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	want := []schemaEndpoint{{Method: "GET", Path: "/v1/apps"}}
	if len(endpoints) != len(want) || endpoints[0] != want[0] {
		t.Fatalf("expected exact endpoint %#v, got %#v", want, endpoints)
	}
}

func TestSchemaSupportsDocumentedDotNotation(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "apps.list"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	want := []schemaEndpoint{{Method: "GET", Path: "/v1/apps"}}
	if len(endpoints) != len(want) || endpoints[0] != want[0] {
		t.Fatalf("expected list endpoint %#v, got %#v", want, endpoints)
	}
}

func TestSchemaDotNotationUsesRelatedResourceCardinality(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "builds.appStoreVersion.get"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	want := []schemaEndpoint{{Method: "GET", Path: "/v1/builds/{id}/appStoreVersion"}}
	if len(endpoints) != len(want) || endpoints[0] != want[0] {
		t.Fatalf("expected to-one endpoint %#v, got %#v", want, endpoints)
	}
}

func TestSchemaDotNotationUsesMetricsResponseCardinality(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "apps.metrics.betaTesterUsages.list"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	want := []schemaEndpoint{{Method: "GET", Path: "/v1/apps/{id}/metrics/betaTesterUsages"}}
	if len(endpoints) != len(want) || endpoints[0] != want[0] {
		t.Fatalf("expected metrics endpoint %#v, got %#v", want, endpoints)
	}
}

func TestSchemaSupportsVersionQualifiedActionDotNotation(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "v2.gameCenterAchievements.get"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	want := []schemaEndpoint{{Method: "GET", Path: "/v2/gameCenterAchievements/{id}"}}
	if len(endpoints) != len(want) || endpoints[0] != want[0] {
		t.Fatalf("expected exact versioned endpoint %#v, got %#v", want, endpoints)
	}
}

func TestSchemaListRejectsPositionalQuery(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "--list", "extra"}, "1.2.3")
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--list does not accept a query") {
		t.Fatalf("expected positional-query diagnostic, got %q", stderr)
	}
}

func TestSchemaRejectsUnknownFlagAfterQuery(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "apps", "--unknown-schema-flag"}, "1.2.3")
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "flag provided but not defined: -unknown-schema-flag") {
		t.Fatalf("expected unknown-flag diagnostic, got %q", stderr)
	}
}

func TestSchemaPreservesArgumentsAfterFlagTerminator(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "--", "--list"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout != "[]\n" {
		t.Fatalf("expected literal query after -- to return an empty JSON array, got %q", stdout)
	}
}

func TestSchemaNoMatchReturnsEmptyJSONArray(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "definitely-no-such-endpoint"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout != "[]\n" {
		t.Fatalf("expected empty JSON array, got %q", stdout)
	}
}

func TestSchemaPreservesFuzzyQueries(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "builds"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var endpoints []schemaEndpoint
	if err := json.Unmarshal([]byte(stdout), &endpoints); err != nil {
		t.Fatalf("unmarshal schema output: %v\nstdout=%s", err, stdout)
	}
	if len(endpoints) < 2 {
		t.Fatalf("expected multiple fuzzy matches, got %#v", endpoints)
	}
	found := false
	for _, endpoint := range endpoints {
		if endpoint.Method == "GET" && endpoint.Path == "/v1/builds" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected GET /v1/builds among fuzzy results, got %#v", endpoints)
	}
}

func TestSchemaFuzzyQueryDoesNotSearchActionSuffixes(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"schema", "update"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout != "[]\n" {
		t.Fatalf("expected path-only fuzzy search to return no matches, got %q", stdout)
	}
}

func TestRun_SchemaInvalidMethodReturnsUsage(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	_, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{"schema", "--list", "--method", "DELTE"}, "1.0.0")
		if code != rootcmd.ExitUsage {
			t.Fatalf("expected exit code %d, got %d", rootcmd.ExitUsage, code)
		}
	})

	if !strings.Contains(stderr, "invalid --method") {
		t.Fatalf("expected invalid --method error, got stderr: %s", stderr)
	}
}

func TestRun_SchemaValidMethodReturnsSuccess(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	stdout, stderr := captureOutput(t, func() {
		code := rootcmd.Run([]string{"schema", "--list", "--method", "DELETE"}, "1.0.0")
		if code != rootcmd.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d", rootcmd.ExitSuccess, code)
		}
	})

	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected empty stderr, got: %s", stderr)
	}
	if !strings.Contains(stdout, "\"method\":\"DELETE\"") {
		t.Fatalf("expected DELETE endpoints in output, got: %s", stdout)
	}
}
