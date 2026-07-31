package install

import (
	"net/url"
	"runtime"
	"strings"
)

// The automatic skills update check launches a detached worker plus a
// PATH-resolved `skills` or `npx` helper, so none of them may inherit the
// caller's credentials. The environment is built from an allowlist rather than
// a secret denylist: an unrecognized variable is dropped instead of judged.
//
// Only variables that executable discovery, the OS loader, and language
// runtimes genuinely need are forwarded.
var skillsCheckSharedEnvAllowlist = []string{
	"HOME",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
	// CA trust-store paths are connectivity configuration: behind a corporate
	// TLS proxy the helpers cannot establish TLS without them. They point at
	// public trust anchors and carry no credential. NODE_OPTIONS stays
	// excluded because it can inject arbitrary code into the helper.
	"NODE_EXTRA_CA_CERTS",
	"PATH",
	"SSL_CERT_DIR",
	"SSL_CERT_FILE",
	"TEMP",
	"TMP",
	"TMPDIR",
	"TZ",
}

var skillsCheckUnixEnvAllowlist = []string{
	"LOGNAME",
	"SHELL",
	"USER",
	"XDG_CACHE_HOME",
	"XDG_CONFIG_HOME",
	"XDG_DATA_HOME",
}

// Proxy configuration is connectivity, not identity: the update check is a
// network operation, so without these variables it silently fails behind a
// corporate proxy. Proxy URLs may embed credentials in their userinfo, query,
// or fragment, and a credential must not cross the helper boundary, so
// HTTP_PROXY and HTTPS_PROXY are reduced to scheme, host, and port before
// forwarding; a value that cannot be provably sanitized is dropped.
// Authenticated proxies therefore still fail closed — the check degrades to
// its cached result rather than forwarding a secret. NO_PROXY is a host list
// with no credential form and is forwarded verbatim. Names are matched
// case-insensitively on every platform because both the upper- and lowercase
// spellings are conventional.
var skillsCheckProxyEnvAllowlist = []string{
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
}

// npm interprets npm_config_* environment configuration case-insensitively,
// and the offline `npx` fallback needs the relocated cache to find the skills
// package. Only the cache location is forwarded: it is a path, not a
// credential, while other npm settings (registry URLs, auth tokens) can carry
// secrets and stay excluded.
var skillsCheckCaseInsensitiveEnvAllowlist = []string{
	"NPM_CONFIG_CACHE",
}

// Windows names are stored uppercase because Windows environment variable names
// are case-insensitive.
var skillsCheckWindowsEnvAllowlist = []string{
	"APPDATA",
	"COMSPEC",
	"HOMEDRIVE",
	"HOMEPATH",
	"LOCALAPPDATA",
	"NUMBER_OF_PROCESSORS",
	"OS",
	"PATHEXT",
	"PROCESSOR_ARCHITECTURE",
	"PROGRAMDATA",
	"PROGRAMFILES",
	"PROGRAMFILES(X86)",
	"PROGRAMW6432",
	"SYSTEMDRIVE",
	"SYSTEMROOT",
	"USERPROFILE",
	"WINDIR",
}

// skillsCheckWorkerEnvironment builds the detached worker environment: the
// allowlisted runtime variables plus the private worker coordination values.
func skillsCheckWorkerEnvironment(base []string, spec skillsCheckWorkerSpec) []string {
	return append(
		filterSkillsCheckEnvironment(base, runtime.GOOS),
		skillsWorkerEnvVar+"=1",
		skillsWorkerCacheEnvVar+"="+spec.cachePath,
		skillsWorkerLockEnvVar+"="+spec.lockPath,
		skillsWorkerTokenEnvVar+"="+spec.token,
	)
}

// skillsCheckHelperEnvironment builds the environment for the PATH-resolved
// `skills` or `npx` helper. Worker coordination values are not allowlisted, so
// they stay inside the worker.
func skillsCheckHelperEnvironment(base []string) []string {
	return filterSkillsCheckEnvironment(base, runtime.GOOS)
}

func filterSkillsCheckEnvironment(base []string, goos string) []string {
	allowed := skillsCheckEnvAllowlistFor(goos)
	filtered := make([]string, 0, len(allowed))
	for _, entry := range base {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if proxyName, isProxy := skillsCheckProxyEnvName(key); isProxy {
			if safe, ok := sanitizeSkillsCheckProxyValue(proxyName, value); ok {
				filtered = append(filtered, key+"="+safe)
			}
			continue
		}
		if skillsCheckEnvNameAllowedCaseInsensitively(key) {
			filtered = append(filtered, entry)
			continue
		}
		if _, ok := allowed[normalizeSkillsCheckEnvKey(key, goos)]; ok {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// skillsCheckEnvNameAllowedCaseInsensitively reports whether key is a
// credential-free setting whose consumer matches names case-insensitively on
// every platform.
func skillsCheckEnvNameAllowedCaseInsensitively(key string) bool {
	upper := strings.ToUpper(key)
	for _, name := range skillsCheckCaseInsensitiveEnvAllowlist {
		if upper == name {
			return true
		}
	}
	return false
}

// skillsCheckProxyEnvName reports whether key names proxy configuration and
// returns its canonical uppercase form.
func skillsCheckProxyEnvName(key string) (string, bool) {
	upper := strings.ToUpper(key)
	for _, name := range skillsCheckProxyEnvAllowlist {
		if upper == name {
			return name, true
		}
	}
	return "", false
}

// sanitizeSkillsCheckProxyValue returns the proxy value safe to forward. A
// hierarchical proxy URL is reduced to its scheme, host, and port: userinfo,
// path, query, and fragment can all carry credentials. The common schemeless
// host:port form is forwarded verbatim only when it contains none of "@", "?",
// "#", or "/", because without those separators the value has no structured
// place for a credential. Anything else is dropped entirely: a credential
// position that cannot be proven safe is not forwarded.
func sanitizeSkillsCheckProxyValue(canonicalName, value string) (string, bool) {
	if canonicalName == "NO_PROXY" {
		return value, true
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value, true
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Host != "" && parsed.Opaque == "" {
		safe := url.URL{Scheme: parsed.Scheme, Host: parsed.Host}
		return safe.String(), true
	}
	if !strings.ContainsAny(trimmed, "@?#/") {
		return value, true
	}
	return "", false
}

func skillsCheckEnvAllowlistFor(goos string) map[string]struct{} {
	platform := skillsCheckUnixEnvAllowlist
	if goos == "windows" {
		platform = skillsCheckWindowsEnvAllowlist
	}

	allowed := make(map[string]struct{}, len(skillsCheckSharedEnvAllowlist)+len(platform))
	for _, name := range skillsCheckSharedEnvAllowlist {
		allowed[normalizeSkillsCheckEnvKey(name, goos)] = struct{}{}
	}
	for _, name := range platform {
		allowed[normalizeSkillsCheckEnvKey(name, goos)] = struct{}{}
	}
	return allowed
}

func normalizeSkillsCheckEnvKey(key, goos string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}
