package distribute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/signing"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

const (
	deviceObservationJSONLimit       = 8 << 20
	deviceObservationDiagnosticLimit = 32 << 10
	deviceObservationMinimumTimeout  = 5 * time.Second
	deviceObservationMaximumTimeout  = 5 * time.Minute
	deviceObservationTimeoutHeadroom = 2 * time.Second
	deviceObservationJSONVersion     = 5
	deviceObservationJSONFilename    = "observation.json"
)

// deviceObservationRequest is intentionally not persisted. In particular, the
// device selector must never enter a distribution plan, run state, or receipt.
type deviceObservationRequest struct {
	DeviceSelector string
	BundleID       string
	Version        string
	Build          string
	Timeout        time.Duration
	Environment    []string
}

// deviceObservation is the complete redacted result exposed by verify. It
// reports only the expected application identity and never the selector,
// device name, UDID, install URL, or devicectl's raw output.
type deviceObservation struct {
	Requested    bool   `json:"requested"`
	DeviceFound  bool   `json:"deviceFound"`
	AppInstalled bool   `json:"appInstalled"`
	BundleID     string `json:"bundleId,omitempty"`
	Version      string `json:"version,omitempty"`
	Build        string `json:"build,omitempty"`
}

// DeviceObservationUnsupportedError reports a platform that cannot perform a
// connected-device observation. It is returned before any child invocation.
type DeviceObservationUnsupportedError struct {
	Platform string
}

func (e *DeviceObservationUnsupportedError) Error() string {
	return fmt.Sprintf("connected-device observation is unsupported on %s; macOS with Xcode devicectl is required", e.Platform)
}

// DeviceObservationDeviceNotFoundError means devicectl could not resolve the
// caller's exact selector to a connected device. The selector is omitted.
type DeviceObservationDeviceNotFoundError struct{}

func (*DeviceObservationDeviceNotFoundError) Error() string {
	return "the requested connected device was not found"
}

// DeviceObservationAppNotInstalledError means the expected bundle identifier
// was not present on the connected device.
type DeviceObservationAppNotInstalledError struct{}

func (*DeviceObservationAppNotInstalledError) Error() string {
	return "the expected app is not installed on the requested device"
}

// DeviceObservationAppMismatchError means the expected bundle exists, but its
// reported version or build is not the verified prepared artifact's value.
type DeviceObservationAppMismatchError struct{}

func (*DeviceObservationAppMismatchError) Error() string {
	return "the installed app version or build does not match the verified prepared artifact"
}

// DeviceObservationCommandError is a redacted devicectl failure. Raw process
// output and the caller's selector are deliberately not retained.
type DeviceObservationCommandError struct {
	Outcome  string
	Code     int
	ExitCode int
}

func (e *DeviceObservationCommandError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("devicectl connected-device observation failed (outcome=%s, code=%d)", e.Outcome, e.Code)
	}
	if e.ExitCode >= 0 {
		return fmt.Sprintf("devicectl connected-device observation failed (exit=%d)", e.ExitCode)
	}
	return "devicectl connected-device observation failed"
}

type deviceObservationRunner interface {
	ResolveDevicectl(context.Context, []string) (string, error)
	Run(context.Context, string, []string, []string) error
}

type processDeviceObservationRunner struct{}

func (processDeviceObservationRunner) ResolveDevicectl(ctx context.Context, environment []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/xcrun", "--find", "devicectl")
	cmd.Env = environment
	stdout := &limitedDeviceObservationBuffer{limit: deviceObservationDiagnosticLimit}
	stderr := &limitedDeviceObservationBuffer{limit: deviceObservationDiagnosticLimit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf("resolve devicectl through the active Xcode toolchain")
	}
	path := strings.TrimSpace(stdout.String())
	if path == "" || !filepath.IsAbs(path) || strings.ContainsAny(path, "\r\n\x00") {
		return "", fmt.Errorf("resolve devicectl through the active Xcode toolchain: invalid absolute path")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("resolve devicectl through the active Xcode toolchain: tool is unavailable")
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("resolve devicectl through the active Xcode toolchain: tool is not a regular executable")
	}
	return filepath.Clean(path), nil
}

func (processDeviceObservationRunner) Run(ctx context.Context, path string, args, environment []string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = environment
	cmd.Stdin = nil
	cmd.Stdout = &limitedDeviceObservationBuffer{limit: deviceObservationDiagnosticLimit}
	cmd.Stderr = &limitedDeviceObservationBuffer{limit: deviceObservationDiagnosticLimit}
	return cmd.Run()
}

type limitedDeviceObservationBuffer struct {
	buffer bytes.Buffer
	limit  int
}

// deviceObservationAfterLstatForTest makes final-component replacement tests
// deterministic. It is nil in production.
var deviceObservationAfterLstatForTest func()

func (b *limitedDeviceObservationBuffer) Write(p []byte) (int, error) {
	written := len(p)
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buffer.Write(p)
	}
	return written, nil
}

func (b *limitedDeviceObservationBuffer) String() string {
	return b.buffer.String()
}

func observeInstalledAppOnDevice(ctx context.Context, request deviceObservationRequest) (deviceObservation, error) {
	return observeInstalledAppOnDeviceWithRunner(ctx, runtime.GOOS, request, processDeviceObservationRunner{})
}

func observeInstalledAppOnDeviceWithRunner(
	ctx context.Context,
	platform string,
	request deviceObservationRequest,
	runner deviceObservationRunner,
) (deviceObservation, error) {
	observation := deviceObservation{Requested: true, BundleID: request.BundleID}
	if platform != "darwin" {
		return observation, &DeviceObservationUnsupportedError{Platform: platform}
	}
	if ctx == nil {
		return observation, fmt.Errorf("connected-device observation context is required")
	}
	if err := validateDeviceObservationRequest(request); err != nil {
		return observation, err
	}
	if runner == nil {
		return observation, fmt.Errorf("connected-device observation runner is required")
	}

	boundedContext, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	environment := signing.SanitizedChildEnvironment(request.Environment)
	if request.Environment == nil {
		environment = signing.SanitizedChildEnvironment(os.Environ())
	}
	devicectlPath, err := runner.ResolveDevicectl(boundedContext, environment)
	if err != nil {
		if contextErr := boundedContext.Err(); contextErr != nil {
			return observation, contextErr
		}
		return observation, fmt.Errorf("resolve devicectl through the active Xcode toolchain")
	}
	if !filepath.IsAbs(devicectlPath) {
		return observation, fmt.Errorf("resolved devicectl path is not absolute")
	}

	tempDir, err := os.MkdirTemp("", "asc-distribute-device-observation-")
	if err != nil {
		return observation, fmt.Errorf("create private devicectl output directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	if err := os.Chmod(tempDir, 0o700); err != nil {
		return observation, fmt.Errorf("protect devicectl output directory: %w", err)
	}
	jsonPath := filepath.Join(tempDir, deviceObservationJSONFilename)
	if _, err := os.Lstat(jsonPath); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return observation, fmt.Errorf("refusing to replace devicectl JSON output")
		}
		return observation, fmt.Errorf("inspect devicectl JSON destination: %w", err)
	}

	toolTimeout := request.Timeout - deviceObservationTimeoutHeadroom
	timeoutSeconds := int64((toolTimeout + time.Second - 1) / time.Second)
	args := []string{
		"--quiet",
		"--timeout", strconv.FormatInt(timeoutSeconds, 10),
		"--json-output", jsonPath,
		"--omit-deprecated-fields-in-json",
		"device", "info", "apps",
		"--device", request.DeviceSelector,
		"--bundle-id", request.BundleID,
	}
	runErr := runner.Run(boundedContext, devicectlPath, args, environment)
	if contextErr := boundedContext.Err(); contextErr != nil {
		return observation, contextErr
	}
	payload, readErr := readDeviceObservationJSON(tempDir)
	if readErr != nil {
		if runErr != nil {
			return observation, redactedDeviceObservationCommandError(runErr)
		}
		return observation, readErr
	}
	envelope, err := parseDeviceObservationEnvelope(payload)
	if err != nil {
		return observation, err
	}
	return interpretDeviceObservation(envelope, request, observation, runErr)
}

func validateDeviceObservationRequest(request deviceObservationRequest) error {
	if err := validateDeviceObservationText("device selector", request.DeviceSelector, 1024); err != nil {
		return err
	}
	if err := validateDeviceObservationText("bundle ID", request.BundleID, 255); err != nil {
		return err
	}
	if err := validateDeviceObservationText("version", request.Version, 255); err != nil {
		return err
	}
	if err := validateDeviceObservationText("build", request.Build, 255); err != nil {
		return err
	}
	if request.Timeout < deviceObservationMinimumTimeout || request.Timeout > deviceObservationMaximumTimeout {
		return fmt.Errorf("device observation timeout must be between %s and %s", deviceObservationMinimumTimeout, deviceObservationMaximumTimeout)
	}
	return nil
}

func validateDeviceObservationText(label, value string, limit int) error {
	if value == "" {
		return fmt.Errorf("%s is required", label)
	}
	if value != strings.TrimSpace(value) {
		return fmt.Errorf("%s must not contain leading or trailing whitespace", label)
	}
	if len(value) > limit {
		return fmt.Errorf("%s exceeds %d bytes", label, limit)
	}
	for _, r := range value {
		if r == 0 || unicode.IsControl(r) {
			return fmt.Errorf("%s contains a control character", label)
		}
	}
	return nil
}

func readDeviceObservationJSON(tempDir string) ([]byte, error) {
	root, err := os.OpenRoot(tempDir)
	if err != nil {
		return nil, fmt.Errorf("open private devicectl output directory: %w", err)
	}
	defer root.Close()
	info, err := root.Lstat(deviceObservationJSONFilename)
	if err != nil {
		return nil, fmt.Errorf("read devicectl JSON output: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("devicectl JSON output is not a regular file")
	}
	if info.Size() > deviceObservationJSONLimit {
		return nil, fmt.Errorf("devicectl JSON output exceeds the %d-byte size limit", deviceObservationJSONLimit)
	}
	if _, nlink, ok := distributionStatIdentity(info); ok && nlink != 1 {
		return nil, fmt.Errorf("devicectl JSON output must not have multiple hard links")
	}
	if deviceObservationAfterLstatForTest != nil {
		deviceObservationAfterLstatForTest()
	}
	file, err := secureopen.OpenExistingNoFollowInRoot(root, deviceObservationJSONFilename)
	if err != nil {
		return nil, fmt.Errorf("open devicectl JSON output: %w", err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect opened devicectl JSON output: %w", err)
	}
	if err := validateStableDeviceObservationFile(info, openedInfo); err != nil {
		return nil, err
	}
	payload, err := io.ReadAll(io.LimitReader(file, deviceObservationJSONLimit+1))
	if err != nil {
		return nil, fmt.Errorf("read devicectl JSON output: %w", err)
	}
	if len(payload) > deviceObservationJSONLimit {
		return nil, fmt.Errorf("devicectl JSON output exceeds the %d-byte size limit", deviceObservationJSONLimit)
	}
	afterRead, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("reinspect devicectl JSON output: %w", err)
	}
	if err := validateStableDeviceObservationFile(openedInfo, afterRead); err != nil {
		return nil, err
	}
	pathAfter, err := root.Lstat(deviceObservationJSONFilename)
	if err != nil {
		return nil, fmt.Errorf("reinspect devicectl JSON path: %w", err)
	}
	if err := validateStableDeviceObservationFile(afterRead, pathAfter); err != nil {
		return nil, err
	}
	return payload, nil
}

func validateStableDeviceObservationFile(expected, actual os.FileInfo) error {
	if !actual.Mode().IsRegular() || actual.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("devicectl JSON output is not a regular file")
	}
	if _, nlink, ok := distributionStatIdentity(actual); ok && nlink != 1 {
		return fmt.Errorf("devicectl JSON output must not have multiple hard links")
	}
	if !os.SameFile(expected, actual) || expected.Mode() != actual.Mode() || expected.Size() != actual.Size() || !expected.ModTime().Equal(actual.ModTime()) {
		return fmt.Errorf("devicectl JSON output changed while being read")
	}
	return nil
}

type deviceObservationEnvelope struct {
	Info   deviceObservationInfo   `json:"info"`
	Result json.RawMessage         `json:"result"`
	Error  *deviceObservationError `json:"error,omitempty"`
}

type deviceObservationInfo struct {
	Arguments   []string                   `json:"arguments"`
	CommandType string                     `json:"commandType"`
	Details     string                     `json:"details,omitempty"`
	Environment map[string]json.RawMessage `json:"environment"`
	JSONVersion int                        `json:"jsonVersion"`
	Outcome     string                     `json:"outcome"`
	Version     string                     `json:"version"`
}

type deviceObservationError struct {
	Code     int             `json:"code"`
	Domain   string          `json:"domain"`
	UserInfo json.RawMessage `json:"userInfo"`
}

type deviceObservationAppsResult struct {
	Apps *[]map[string]json.RawMessage `json:"apps"`
}

func parseDeviceObservationEnvelope(payload []byte) (deviceObservationEnvelope, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return deviceObservationEnvelope{}, fmt.Errorf("devicectl JSON output is empty")
	}
	if err := rejectDuplicateDeviceObservationJSONKeys(payload); err != nil {
		return deviceObservationEnvelope{}, fmt.Errorf("parse devicectl JSON output: invalid JSON structure")
	}
	var envelope deviceObservationEnvelope
	if err := decodeStrictDeviceObservationJSON(payload, &envelope); err != nil {
		return deviceObservationEnvelope{}, fmt.Errorf("parse devicectl JSON output: invalid schema")
	}
	if envelope.Info.CommandType != "devicectl.device.info.apps" {
		return deviceObservationEnvelope{}, fmt.Errorf("parse devicectl JSON output: unexpected command type")
	}
	if envelope.Info.JSONVersion != deviceObservationJSONVersion {
		return deviceObservationEnvelope{}, fmt.Errorf("parse devicectl JSON output: unsupported JSON schema version %d", envelope.Info.JSONVersion)
	}
	if envelope.Info.Outcome == "" {
		return deviceObservationEnvelope{}, fmt.Errorf("parse devicectl JSON output: outcome is required")
	}
	return envelope, nil
}

func decodeStrictDeviceObservationJSON(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func rejectDuplicateDeviceObservationJSONKeys(payload []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := scanDeviceObservationJSONValue(decoder); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected trailing JSON token %v", token)
	}
	return nil
}

func scanDeviceObservationJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key")
			}
			seen[key] = struct{}{}
			if err := scanDeviceObservationJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := scanDeviceObservationJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array is not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func interpretDeviceObservation(
	envelope deviceObservationEnvelope,
	request deviceObservationRequest,
	observation deviceObservation,
	runErr error,
) (deviceObservation, error) {
	switch envelope.Info.Outcome {
	case "failed":
		if envelope.Error != nil && envelope.Error.Domain == "com.apple.dt.CoreDeviceError" && envelope.Error.Code == 1000 {
			return observation, &DeviceObservationDeviceNotFoundError{}
		}
		return observation, structuredDeviceObservationCommandError(envelope, runErr)
	case "timeout":
		return observation, structuredDeviceObservationCommandError(envelope, runErr)
	case "success":
		if runErr != nil {
			return observation, redactedDeviceObservationCommandError(runErr)
		}
	default:
		return observation, fmt.Errorf("parse devicectl JSON output: unsupported outcome")
	}

	observation.DeviceFound = true
	if len(bytes.TrimSpace(envelope.Result)) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Result), []byte("null")) {
		return observation, fmt.Errorf("parse devicectl JSON output: successful result is missing")
	}
	var result deviceObservationAppsResult
	if err := decodeStrictDeviceObservationJSON(envelope.Result, &result); err != nil {
		return observation, fmt.Errorf("parse devicectl apps result: invalid schema")
	}
	if result.Apps == nil {
		return observation, fmt.Errorf("parse devicectl apps result: apps are required")
	}
	var matched bool
	var actualVersion string
	var actualBuild string
	for _, app := range *result.Apps {
		bundleID, err := requiredDeviceObservationString(app, "bundleIdentifier")
		if err != nil {
			return observation, err
		}
		if bundleID != request.BundleID {
			return observation, fmt.Errorf("parse devicectl apps result: query returned an unexpected bundle identifier")
		}
		if matched {
			return observation, fmt.Errorf("parse devicectl apps result: duplicate matching app")
		}
		matched = true
		observation.AppInstalled = true
		observation.BundleID = bundleID
		actualVersion, err = requiredDeviceObservationString(app, "version")
		if err != nil {
			return observation, err
		}
		actualBuild, err = requiredDeviceObservationString(app, "bundleVersion")
		if err != nil {
			return observation, err
		}
	}
	if !matched {
		return observation, &DeviceObservationAppNotInstalledError{}
	}
	if actualVersion != request.Version || actualBuild != request.Build {
		return observation, &DeviceObservationAppMismatchError{}
	}
	observation.Version = request.Version
	observation.Build = request.Build
	return observation, nil
}

func requiredDeviceObservationString(values map[string]json.RawMessage, key string) (string, error) {
	raw, ok := values[key]
	if !ok {
		return "", fmt.Errorf("parse devicectl apps result: %s is missing", key)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("parse devicectl apps result: %s must be a string", key)
	}
	if err := validateDeviceObservationText(key, value, 1024); err != nil {
		return "", fmt.Errorf("parse devicectl apps result: %w", err)
	}
	return value, nil
}

func structuredDeviceObservationCommandError(envelope deviceObservationEnvelope, runErr error) error {
	errorValue := &DeviceObservationCommandError{Outcome: envelope.Info.Outcome, ExitCode: exitCodeForDeviceObservation(runErr)}
	if envelope.Error != nil {
		errorValue.Code = envelope.Error.Code
	}
	return errorValue
}

func redactedDeviceObservationCommandError(runErr error) error {
	return &DeviceObservationCommandError{ExitCode: exitCodeForDeviceObservation(runErr)}
}

func exitCodeForDeviceObservation(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}
