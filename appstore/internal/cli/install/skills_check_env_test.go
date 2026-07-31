package install

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Credentials a real user plausibly exports before running asc. None of them
// belong to a PATH-resolved skills or npx helper.
var skillsCheckCredentialSentinels = map[string]string{
	"ASC_PRIVATE_KEY":                    "asc-red-sentinel-private-key-2ba917",
	"ASC_KEY_ID":                         "asc-red-sentinel-key-id-8cd034",
	"ASC_ISSUER_ID":                      "asc-red-sentinel-issuer-6f1e52",
	"ASC_PRIVATE_KEY_PATH":               "/tmp/asc-red-sentinel-key-path-4a70bd.p8",
	"ASC_SLACK_WEBHOOK":                  "https://hooks.slack.com/services/asc-red-sentinel-webhook-19e6cf",
	"GITHUB_TOKEN":                       "asc-red-sentinel-github-token-d5b283",
	"MATCH_PASSWORD":                     "asc-red-sentinel-signing-pw-7e40ca",
	"NPM_TOKEN":                          "asc-red-sentinel-npm-token-b921fe",
	"AWS_SECRET_ACCESS_KEY":              "asc-red-sentinel-aws-secret-3cd158",
	"SOME_UNRECOGNIZED_INTERNAL_SETTING": "asc-red-sentinel-unknown-var-0f7a62",
}

func seedSkillsCheckCredentials(t *testing.T) {
	t.Helper()
	for key, value := range skillsCheckCredentialSentinels {
		t.Setenv(key, value)
	}
}

func assertNoCredentialSentinels(t *testing.T, label string, env []string) {
	t.Helper()
	joined := strings.Join(env, "\n")
	for key, value := range skillsCheckCredentialSentinels {
		if strings.Contains(joined, value) {
			t.Fatalf("%s received credential %s: %s", label, key, joined)
		}
	}
}

func TestSkillsCheckWorkerEnvironmentExcludesCredentials(t *testing.T) {
	seedSkillsCheckCredentials(t)

	env := skillsCheckWorkerEnvironment(os.Environ(), skillsCheckWorkerSpec{
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
		lockPath:  filepath.Join(t.TempDir(), "lock"),
		token:     "worker-token",
	})

	assertNoCredentialSentinels(t, "detached worker", env)

	values := envMap(env)
	if values["PATH"] != os.Getenv("PATH") {
		t.Fatalf("worker PATH = %q, want the parent PATH", values["PATH"])
	}
	if values[skillsWorkerEnvVar] != "1" {
		t.Fatalf("worker coordination variable %s = %q, want 1", skillsWorkerEnvVar, values[skillsWorkerEnvVar])
	}
	if values[skillsWorkerTokenEnvVar] != "worker-token" {
		t.Fatalf("worker token = %q, want worker-token", values[skillsWorkerTokenEnvVar])
	}
}

func TestSkillsCheckHelperEnvironmentExcludesWorkerCoordinationVariables(t *testing.T) {
	seedSkillsCheckCredentials(t)
	t.Setenv(skillsWorkerEnvVar, "1")
	t.Setenv(skillsWorkerTokenEnvVar, "worker-token")

	env := skillsCheckHelperEnvironment(os.Environ())

	assertNoCredentialSentinels(t, "skills helper", env)

	values := envMap(env)
	if _, ok := values[skillsWorkerEnvVar]; ok {
		t.Fatalf("helper env exposed %s", skillsWorkerEnvVar)
	}
	if _, ok := values[skillsWorkerTokenEnvVar]; ok {
		t.Fatalf("helper env exposed %s", skillsWorkerTokenEnvVar)
	}
	if values["PATH"] != os.Getenv("PATH") {
		t.Fatal("helper env dropped PATH, breaking executable discovery")
	}
}

func TestDefaultRunSkillsCheckCommandGivesDirectHelperOnlyAllowlistedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	seedSkillsCheckCredentials(t)

	mockSkills := writeExecutable(t, "#!/bin/sh\nenv\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return mockSkills, nil
	}
	lookupExecutable = func(string) (string, error) {
		t.Fatal("lookupExecutable should not run when skills is available")
		return "", nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}

	assertNoCredentialSentinels(t, "direct skills helper", strings.Split(output, "\n"))
	if !strings.Contains(output, "PATH=") {
		t.Fatalf("direct skills helper lost PATH: %q", output)
	}
}

func TestDefaultRunSkillsCheckCommandGivesNpxFallbackOnlyAllowlistedEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	seedSkillsCheckCredentials(t)

	mockNpx := writeExecutable(t, "#!/bin/sh\nenv\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return "", os.ErrNotExist
	}
	lookupExecutable = func(string) (string, error) {
		return mockNpx, nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}

	assertNoCredentialSentinels(t, "npx fallback", strings.Split(output, "\n"))
	if !strings.Contains(output, "npm_config_offline=true") {
		t.Fatalf("npx fallback lost the offline cache setting: %q", output)
	}
	if !strings.Contains(output, "PATH=") {
		t.Fatalf("npx fallback lost PATH: %q", output)
	}
}

func TestSkillsCheckEnvironmentAllowlistIsPlatformAware(t *testing.T) {
	unix := skillsCheckEnvAllowlistFor("darwin")
	windows := skillsCheckEnvAllowlistFor("windows")

	for _, name := range []string{"PATH", "HOME", "TMPDIR"} {
		if _, ok := unix[name]; !ok {
			t.Fatalf("unix allowlist is missing %s", name)
		}
	}
	for _, name := range []string{"PATH", "SYSTEMROOT", "PATHEXT", "USERPROFILE", "APPDATA"} {
		if _, ok := windows[name]; !ok {
			t.Fatalf("windows allowlist is missing %s", name)
		}
	}
	if _, ok := windows["SHELL"]; ok {
		t.Fatal("windows allowlist should not carry Unix-only variables")
	}
	if _, ok := unix["SYSTEMROOT"]; ok {
		t.Fatal("unix allowlist should not carry Windows-only variables")
	}
}

func TestFilterSkillsCheckEnvironmentMatchesWindowsCaseInsensitively(t *testing.T) {
	base := []string{
		"SystemRoot=C:\\Windows",
		"Path=C:\\Windows\\System32",
		"NPM_TOKEN=" + skillsCheckCredentialSentinels["NPM_TOKEN"],
		"malformed-entry",
	}

	filtered := filterSkillsCheckEnvironment(base, "windows")

	values := envMap(filtered)
	if values["SystemRoot"] != "C:\\Windows" {
		t.Fatalf("windows filter dropped SystemRoot: %v", filtered)
	}
	if values["Path"] != "C:\\Windows\\System32" {
		t.Fatalf("windows filter dropped Path: %v", filtered)
	}
	assertNoCredentialSentinels(t, "windows filter", filtered)
	if len(filtered) != 2 {
		t.Fatalf("windows filter kept unexpected entries: %v", filtered)
	}
}

func TestSkillsCheckEnvironmentForwardsProxyConfigurationWithoutCredentials(t *testing.T) {
	const proxyPasswordSentinel = "asc-red-sentinel-proxy-pw-5d11c9"
	base := []string{
		"HTTP_PROXY=http://proxy.corp.example:3128",
		"https_proxy=https://scanner:" + proxyPasswordSentinel + "@proxy.corp.example:3128",
		"no_proxy=localhost,127.0.0.1,.corp.example",
		"ALL_PROXY=socks5://scanner:" + proxyPasswordSentinel + "@proxy.corp.example:1080",
		"NPM_TOKEN=" + skillsCheckCredentialSentinels["NPM_TOKEN"],
	}

	filtered := filterSkillsCheckEnvironment(base, "darwin")
	values := envMap(filtered)

	if values["HTTP_PROXY"] != "http://proxy.corp.example:3128" {
		t.Fatalf("HTTP_PROXY = %q, want it forwarded verbatim", values["HTTP_PROXY"])
	}
	if values["https_proxy"] != "https://proxy.corp.example:3128" {
		t.Fatalf("https_proxy = %q, want the proxy URL with its userinfo stripped", values["https_proxy"])
	}
	if values["no_proxy"] != "localhost,127.0.0.1,.corp.example" {
		t.Fatalf("no_proxy = %q, want the host list forwarded verbatim", values["no_proxy"])
	}
	if _, ok := values["ALL_PROXY"]; ok {
		t.Fatalf("ALL_PROXY should stay outside the allowlist: %v", filtered)
	}
	joined := strings.Join(filtered, "\n")
	if strings.Contains(joined, proxyPasswordSentinel) {
		t.Fatalf("proxy credential crossed the helper boundary: %v", filtered)
	}
	assertNoCredentialSentinels(t, "proxy-aware filter", filtered)
}

func TestSkillsCheckEnvironmentStripsProxyQueryAndFragmentCredentials(t *testing.T) {
	const proxyTokenSentinel = "asc-red-sentinel-proxy-token-9b21e4"
	base := []string{
		// No userinfo, but the query and fragment can still carry secrets.
		"HTTP_PROXY=https://proxy.corp.example:3128/?token=" + proxyTokenSentinel,
		"HTTPS_PROXY=https://proxy.corp.example:3128#" + proxyTokenSentinel,
		// The schemeless host:port form is common and has no place to hide a
		// structured credential.
		"http_proxy=proxy.corp.example:3128",
	}

	filtered := filterSkillsCheckEnvironment(base, "darwin")

	values := envMap(filtered)
	if values["HTTP_PROXY"] != "https://proxy.corp.example:3128" {
		t.Fatalf("HTTP_PROXY = %q, want scheme and host only", values["HTTP_PROXY"])
	}
	if values["HTTPS_PROXY"] != "https://proxy.corp.example:3128" {
		t.Fatalf("HTTPS_PROXY = %q, want scheme and host only", values["HTTPS_PROXY"])
	}
	if values["http_proxy"] != "proxy.corp.example:3128" {
		t.Fatalf("http_proxy = %q, want the schemeless host:port forwarded verbatim", values["http_proxy"])
	}
	if strings.Contains(strings.Join(filtered, "\n"), proxyTokenSentinel) {
		t.Fatalf("proxy credential crossed the helper boundary: %v", filtered)
	}
}

func TestSkillsCheckEnvironmentDropsProxyValuesItCannotSanitize(t *testing.T) {
	const proxyPasswordSentinel = "asc-red-sentinel-proxy-pw-5d11c9"
	base := []string{
		// No scheme, so the credential position cannot be proven; the value
		// must be dropped rather than forwarded.
		"HTTP_PROXY=scanner:" + proxyPasswordSentinel + "@proxy.corp.example:3128",
		// A schemeless value with a path segment can hide a secret there.
		"HTTPS_PROXY=proxy.corp.example:3128/" + proxyPasswordSentinel,
	}

	filtered := filterSkillsCheckEnvironment(base, "darwin")

	if len(filtered) != 0 {
		t.Fatalf("unsanitizable proxy value was forwarded: %v", filtered)
	}
}

func TestSkillsCheckEnvironmentForwardsProxyConfigurationOnWindows(t *testing.T) {
	const proxyPasswordSentinel = "asc-red-sentinel-proxy-pw-5d11c9"
	base := []string{
		"Http_Proxy=http://scanner:" + proxyPasswordSentinel + "@proxy.corp.example:3128",
	}

	filtered := filterSkillsCheckEnvironment(base, "windows")

	values := envMap(filtered)
	if values["Http_Proxy"] != "http://proxy.corp.example:3128" {
		t.Fatalf("Http_Proxy = %q, want the proxy URL with its userinfo stripped", values["Http_Proxy"])
	}
	if strings.Contains(strings.Join(filtered, "\n"), proxyPasswordSentinel) {
		t.Fatalf("proxy credential crossed the helper boundary: %v", filtered)
	}
}

func TestSkillsCheckEnvironmentForwardsCATrustPaths(t *testing.T) {
	base := []string{
		"SSL_CERT_FILE=/etc/corp/ca-bundle.pem",
		"SSL_CERT_DIR=/etc/corp/certs",
		"NODE_EXTRA_CA_CERTS=/etc/corp/extra-ca.pem",
		// NODE_OPTIONS can inject arbitrary code into the helper and must not
		// ride along with the trust-store paths.
		"NODE_OPTIONS=--require=/tmp/evil.js",
	}

	filtered := filterSkillsCheckEnvironment(base, "darwin")

	values := envMap(filtered)
	if values["SSL_CERT_FILE"] != "/etc/corp/ca-bundle.pem" {
		t.Fatalf("SSL_CERT_FILE = %q, want the trust-store path forwarded", values["SSL_CERT_FILE"])
	}
	if values["SSL_CERT_DIR"] != "/etc/corp/certs" {
		t.Fatalf("SSL_CERT_DIR = %q, want the trust-store path forwarded", values["SSL_CERT_DIR"])
	}
	if values["NODE_EXTRA_CA_CERTS"] != "/etc/corp/extra-ca.pem" {
		t.Fatalf("NODE_EXTRA_CA_CERTS = %q, want the trust-store path forwarded", values["NODE_EXTRA_CA_CERTS"])
	}
	if _, ok := values["NODE_OPTIONS"]; ok {
		t.Fatalf("NODE_OPTIONS must stay outside the allowlist: %v", filtered)
	}
}

func TestSkillsCheckEnvironmentForwardsNpmCacheLocation(t *testing.T) {
	base := []string{
		"npm_config_cache=/custom/npm-cache",
		"NPM_CONFIG_CACHE=/custom/npm-cache-upper",
		// A registry URL can embed an access token, so it must not ride along
		// with the cache location.
		"npm_config_registry=https://" + skillsCheckCredentialSentinels["NPM_TOKEN"] + "@registry.corp.example",
		"npm_config__authToken=" + skillsCheckCredentialSentinels["NPM_TOKEN"],
	}

	filtered := filterSkillsCheckEnvironment(base, "darwin")

	values := envMap(filtered)
	if values["npm_config_cache"] != "/custom/npm-cache" {
		t.Fatalf("npm_config_cache = %q, want the relocated cache forwarded for the offline npx fallback", values["npm_config_cache"])
	}
	if values["NPM_CONFIG_CACHE"] != "/custom/npm-cache-upper" {
		t.Fatalf("NPM_CONFIG_CACHE = %q, want npm's case-insensitive spelling forwarded too", values["NPM_CONFIG_CACHE"])
	}
	if len(filtered) != 2 {
		t.Fatalf("only the cache location may be forwarded, got: %v", filtered)
	}
	assertNoCredentialSentinels(t, "npm cache filter", filtered)
}

func TestSkillsCheckWorkerEnvironmentForwardsProxyConfiguration(t *testing.T) {
	const proxyPasswordSentinel = "asc-red-sentinel-proxy-pw-5d11c9"
	t.Setenv("HTTP_PROXY", "http://scanner:"+proxyPasswordSentinel+"@proxy.corp.example:3128")
	seedSkillsCheckCredentials(t)

	env := skillsCheckWorkerEnvironment(os.Environ(), skillsCheckWorkerSpec{
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
		lockPath:  filepath.Join(t.TempDir(), "lock"),
		token:     "worker-token",
	})

	values := envMap(env)
	if values["HTTP_PROXY"] != "http://proxy.corp.example:3128" {
		t.Fatalf("worker HTTP_PROXY = %q, want the proxy URL with its userinfo stripped", values["HTTP_PROXY"])
	}
	if strings.Contains(strings.Join(env, "\n"), proxyPasswordSentinel) {
		t.Fatalf("proxy credential reached the detached worker: %v", env)
	}
	assertNoCredentialSentinels(t, "proxy-aware worker", env)
}

func envMap(env []string) map[string]string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	return values
}
