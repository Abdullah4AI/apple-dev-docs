package distribute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

const deviceObservationFixture = `{
  "info": {
    "arguments": ["devicectl", "--quiet"],
    "commandType": "devicectl.device.info.apps",
    "environment": {"TERM": "dumb"},
    "jsonVersion": 5,
    "outcome": "success",
    "version": "642.9.1"
  },
  "result": {
    "apps": [{
      "appClip": false,
      "bundleIdentifier": "com.example.agent",
      "bundleVersion": "42",
      "name": "Agent App",
      "url": "file:///private/var/containers/Bundle/Application/redacted/Agent.app",
      "version": "1.2.3"
    }]
  }
}`

type fakeDeviceObservationRunner struct {
	resolvedPath string
	fixture      string
	runErr       error
	resolveCalls int
	runCalls     int
	path         string
	args         []string
	environment  []string
	jsonPath     string
	pathExisted  bool
	parentMode   os.FileMode
}

func (r *fakeDeviceObservationRunner) ResolveDevicectl(context.Context, []string) (string, error) {
	r.resolveCalls++
	if r.resolvedPath == "" {
		return "/Applications/Xcode.app/Contents/Developer/usr/bin/devicectl", nil
	}
	return r.resolvedPath, nil
}

func (r *fakeDeviceObservationRunner) Run(_ context.Context, path string, args, environment []string) error {
	r.runCalls++
	r.path = path
	r.args = append([]string(nil), args...)
	r.environment = append([]string(nil), environment...)
	for i, arg := range args {
		if arg == "--json-output" && i+1 < len(args) {
			r.jsonPath = args[i+1]
			break
		}
	}
	if r.jsonPath == "" {
		return fmt.Errorf("test runner did not receive --json-output")
	}
	if _, err := os.Lstat(r.jsonPath); err == nil {
		r.pathExisted = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	info, err := os.Stat(filepath.Dir(r.jsonPath))
	if err != nil {
		return err
	}
	r.parentMode = info.Mode().Perm()
	if r.fixture != "" {
		if err := os.WriteFile(r.jsonPath, []byte(r.fixture), 0o644); err != nil {
			return err
		}
	}
	return r.runErr
}

func validDeviceObservationRequest() deviceObservationRequest {
	return deviceObservationRequest{
		DeviceSelector: "Rudrank 16 Pro Max",
		BundleID:       "com.example.agent",
		Version:        "1.2.3",
		Build:          "42",
		Timeout:        30 * time.Second,
		Environment: []string{
			"PATH=/usr/bin:/bin",
			"HOME=/Users/example",
			"DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer",
			"ASC_PRIVATE_KEY=must-not-leak",
			"AWS_SECRET_ACCESS_KEY=must-not-leak",
		},
	}
}

func TestObserveInstalledAppOnDeviceUsesExactReadOnlyDevicectlQuery(t *testing.T) {
	runner := &fakeDeviceObservationRunner{fixture: deviceObservationFixture}
	request := validDeviceObservationRequest()

	got, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", request, runner)
	if err != nil {
		t.Fatalf("observeInstalledAppOnDeviceWithRunner() error = %v", err)
	}
	want := deviceObservation{
		Requested:    true,
		DeviceFound:  true,
		AppInstalled: true,
		BundleID:     "com.example.agent",
		Version:      "1.2.3",
		Build:        "42",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("observation = %#v, want %#v", got, want)
	}
	wantArgsPrefix := []string{
		"--quiet",
		"--timeout", "28",
		"--json-output",
	}
	if len(runner.args) < len(wantArgsPrefix)+1 || !reflect.DeepEqual(runner.args[:len(wantArgsPrefix)], wantArgsPrefix) {
		t.Fatalf("devicectl args = %q, want prefix %q", runner.args, wantArgsPrefix)
	}
	wantSuffix := []string{
		"--omit-deprecated-fields-in-json",
		"device", "info", "apps",
		"--device", request.DeviceSelector,
		"--bundle-id", request.BundleID,
	}
	if !reflect.DeepEqual(runner.args[len(wantArgsPrefix)+1:], wantSuffix) {
		t.Fatalf("devicectl args suffix = %q, want %q", runner.args[len(wantArgsPrefix)+1:], wantSuffix)
	}
	if runner.path != "/Applications/Xcode.app/Contents/Developer/usr/bin/devicectl" {
		t.Fatalf("devicectl path = %q", runner.path)
	}
	if runner.pathExisted {
		t.Fatal("JSON destination existed before devicectl invocation; want create-only path")
	}
	if runner.parentMode != 0o700 {
		t.Fatalf("JSON parent mode = %04o, want 0700", runner.parentMode)
	}
	if _, err := os.Lstat(filepath.Dir(runner.jsonPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private temp directory remains after observation: %v", err)
	}
	joinedEnv := strings.Join(runner.environment, "\n")
	if strings.Contains(joinedEnv, "must-not-leak") || strings.Contains(joinedEnv, "AWS_SECRET_ACCESS_KEY") || strings.Contains(joinedEnv, "ASC_PRIVATE_KEY") {
		t.Fatalf("secret environment leaked to devicectl: %q", runner.environment)
	}
	for _, disallowed := range []string{"install", "uninstall", "launch", "process"} {
		if slices.Contains(runner.args, disallowed) {
			t.Fatalf("read-only devicectl query contains %q: %q", disallowed, runner.args)
		}
	}
}

func TestObserveInstalledAppOnDeviceUnsupportedPlatformStopsBeforeInvocation(t *testing.T) {
	runner := &fakeDeviceObservationRunner{fixture: deviceObservationFixture}
	got, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "linux", validDeviceObservationRequest(), runner)
	var unsupported *DeviceObservationUnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want DeviceObservationUnsupportedError", err)
	}
	if !got.Requested || got.DeviceFound || got.AppInstalled {
		t.Fatalf("observation = %#v", got)
	}
	if runner.resolveCalls != 0 || runner.runCalls != 0 {
		t.Fatalf("runner called on unsupported platform: resolve=%d run=%d", runner.resolveCalls, runner.runCalls)
	}
}

func TestObserveInstalledAppOnDeviceRejectsInvalidRequestBeforeInvocation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*deviceObservationRequest)
	}{
		{name: "empty selector", mutate: func(r *deviceObservationRequest) { r.DeviceSelector = "" }},
		{name: "trimmed selector", mutate: func(r *deviceObservationRequest) { r.DeviceSelector = " device " }},
		{name: "empty bundle", mutate: func(r *deviceObservationRequest) { r.BundleID = "" }},
		{name: "control byte", mutate: func(r *deviceObservationRequest) { r.Version = "1.2\n3" }},
		{name: "empty build", mutate: func(r *deviceObservationRequest) { r.Build = "" }},
		{name: "too short timeout", mutate: func(r *deviceObservationRequest) { r.Timeout = 4 * time.Second }},
		{name: "too long timeout", mutate: func(r *deviceObservationRequest) { r.Timeout = 6 * time.Minute }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeDeviceObservationRunner{fixture: deviceObservationFixture}
			request := validDeviceObservationRequest()
			test.mutate(&request)
			if _, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", request, runner); err == nil {
				t.Fatal("expected validation error")
			}
			if runner.resolveCalls != 0 || runner.runCalls != 0 {
				t.Fatalf("runner called after validation error: resolve=%d run=%d", runner.resolveCalls, runner.runCalls)
			}
		})
	}
}

func TestObserveInstalledAppOnDeviceReportsAbsentDevice(t *testing.T) {
	fixture := `{
		"info":{"arguments":[],"commandType":"devicectl.device.info.apps","environment":{},"jsonVersion":5,"outcome":"failed","version":"642.9.1"},
		"result":null,
		"error":{"code":1000,"domain":"com.apple.dt.CoreDeviceError","userInfo":{}}
	}`
	runner := &fakeDeviceObservationRunner{fixture: fixture, runErr: errors.New("SECRET-RAW-OUTPUT")}
	got, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner)
	var notFound *DeviceObservationDeviceNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want DeviceObservationDeviceNotFoundError", err)
	}
	if strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("error leaked raw child output: %v", err)
	}
	if !got.Requested || got.DeviceFound || got.AppInstalled {
		t.Fatalf("observation = %#v", got)
	}
}

func TestObserveInstalledAppOnDeviceRedactsUntrustedErrorDomain(t *testing.T) {
	selector := "secret-device-selector"
	fixture := `{
		"info":{"arguments":[],"commandType":"devicectl.device.info.apps","environment":{},"jsonVersion":5,"outcome":"failed","version":"642.9.1"},
		"result":null,
		"error":{"code":77,"domain":"evil\nsecret-device-selector","userInfo":{"raw":"secret-device-selector"}}
	}`
	runner := &fakeDeviceObservationRunner{fixture: fixture, runErr: errors.New("raw child error")}
	request := validDeviceObservationRequest()
	request.DeviceSelector = selector
	_, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", request, runner)
	var commandError *DeviceObservationCommandError
	if !errors.As(err, &commandError) {
		t.Fatalf("error = %v, want DeviceObservationCommandError", err)
	}
	if strings.Contains(err.Error(), selector) || strings.ContainsAny(err.Error(), "\r\n") || strings.Contains(err.Error(), "evil") {
		t.Fatalf("error leaked untrusted devicectl fields: %q", err)
	}
}

func TestObserveInstalledAppOnDeviceReportsAppAbsent(t *testing.T) {
	fixture := `{
		"info":{"arguments":[],"commandType":"devicectl.device.info.apps","environment":{},"jsonVersion":5,"outcome":"success","version":"642.9.1"},
		"result":{"apps":[]}
	}`
	runner := &fakeDeviceObservationRunner{fixture: fixture}
	got, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner)
	var notInstalled *DeviceObservationAppNotInstalledError
	if !errors.As(err, &notInstalled) {
		t.Fatalf("error = %v, want DeviceObservationAppNotInstalledError", err)
	}
	if !got.Requested || !got.DeviceFound || got.AppInstalled || got.BundleID != "com.example.agent" {
		t.Fatalf("observation = %#v", got)
	}
}

func TestObserveInstalledAppOnDeviceReportsExactVersionMismatch(t *testing.T) {
	fixture := strings.Replace(deviceObservationFixture, `"version": "1.2.3"`, `"version": "1.2.4"`, 1)
	runner := &fakeDeviceObservationRunner{fixture: fixture}
	got, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner)
	var mismatch *DeviceObservationAppMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("error = %v, want DeviceObservationAppMismatchError", err)
	}
	if got.Version != "" || got.Build != "" || !got.AppInstalled {
		t.Fatalf("observation = %#v", got)
	}
}

func TestObserveInstalledAppOnDeviceRedactsUntrustedMismatchValues(t *testing.T) {
	secret := "secret-device-selector"
	fixture := strings.Replace(deviceObservationFixture, `"version": "1.2.3"`, `"version": "`+secret+`"`, 1)
	runner := &fakeDeviceObservationRunner{fixture: fixture}
	request := validDeviceObservationRequest()
	request.DeviceSelector = secret
	got, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", request, runner)
	var mismatch *DeviceObservationAppMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("error = %v, want DeviceObservationAppMismatchError", err)
	}
	if strings.Contains(fmt.Sprintf("%#v", got), secret) || strings.Contains(fmt.Sprintf("%#v", mismatch), secret) || strings.Contains(err.Error(), secret) {
		t.Fatalf("mismatch retained untrusted selector: observation=%#v error=%#v", got, mismatch)
	}
}

func TestObserveInstalledAppOnDeviceStrictlyRejectsInvalidJSON(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{name: "trailing value", fixture: deviceObservationFixture + `{}`},
		{name: "duplicate key", fixture: strings.Replace(deviceObservationFixture, `"outcome": "success"`, `"outcome": "success", "outcome": "failed"`, 1)},
		{name: "wrong command", fixture: strings.Replace(deviceObservationFixture, "devicectl.device.info.apps", "devicectl.device.install.app", 1)},
		{name: "future schema", fixture: strings.Replace(deviceObservationFixture, `"jsonVersion": 5`, `"jsonVersion": 6`, 1)},
		{name: "wrong field type", fixture: strings.Replace(deviceObservationFixture, `"bundleVersion": "42"`, `"bundleVersion": 42`, 1)},
		{name: "missing apps", fixture: strings.Replace(deviceObservationFixture, `"apps": [{`, `"unexpectedSecretField": [{`, 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeDeviceObservationRunner{fixture: test.fixture}
			if _, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner); err == nil {
				t.Fatal("expected strict JSON error")
			} else if strings.Contains(err.Error(), "unexpectedSecretField") {
				t.Fatalf("parse error leaked raw tool output: %v", err)
			}
		})
	}
}

func TestObserveInstalledAppOnDeviceRejectsOversizedJSON(t *testing.T) {
	fixture := strings.Repeat("x", deviceObservationJSONLimit+1)
	runner := &fakeDeviceObservationRunner{fixture: fixture}
	if _, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("error = %v, want size limit", err)
	}
}

func TestObserveInstalledAppOnDeviceRejectsFinalSymlinkSwap(t *testing.T) {
	originalHook := deviceObservationAfterLstatForTest
	defer func() { deviceObservationAfterLstatForTest = originalHook }()
	runner := &fakeDeviceObservationRunner{fixture: deviceObservationFixture}
	deviceObservationAfterLstatForTest = func() {
		// The hook executes after the trusted Lstat and before the no-follow open.
		// A same-UID attacker replaces the pathname with a link to different JSON.
		path := runner.jsonPath
		moved := path + ".moved"
		if err := os.Rename(path, moved); err != nil {
			t.Fatalf("rename observation output: %v", err)
		}
		if err := os.Symlink(filepath.Base(moved), path); err != nil {
			t.Fatalf("replace observation output with symlink: %v", err)
		}
	}
	if _, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner); err == nil {
		t.Fatal("expected final-symlink replacement rejection")
	}
}

func TestObserveInstalledAppOnDeviceRejectsLateHardLink(t *testing.T) {
	originalHook := deviceObservationAfterLstatForTest
	defer func() { deviceObservationAfterLstatForTest = originalHook }()
	runner := &fakeDeviceObservationRunner{fixture: deviceObservationFixture}
	deviceObservationAfterLstatForTest = func() {
		if err := os.Link(runner.jsonPath, runner.jsonPath+".link"); err != nil {
			t.Skipf("hard links unavailable: %v", err)
		}
	}
	if _, err := observeInstalledAppOnDeviceWithRunner(context.Background(), "darwin", validDeviceObservationRequest(), runner); err == nil || !strings.Contains(err.Error(), "hard link") {
		t.Fatalf("error = %v, want late hard-link rejection", err)
	}
}
