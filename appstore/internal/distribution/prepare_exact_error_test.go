package distribution

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"howett.net/plist"
)

func TestPrepareIPAPathExactClassifiesDeterministicExactContentFailuresAsNotEligible(t *testing.T) {
	tests := []struct {
		name      string
		write     func(*testing.T) string
		configure func(*testing.T)
	}{
		{
			name: "invalid ZIP",
			write: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "Invalid.ipa")
				if err := os.WriteFile(path, []byte("not a ZIP archive"), 0o600); err != nil {
					t.Fatal(err)
				}
				return path
			},
		},
		{
			name: "unsafe ZIP entry",
			write: func(t *testing.T) string {
				return writeExactErrorIPA(t, []exactErrorZIPEntry{
					{name: "../escape", data: []byte("unsafe")},
					{name: "Payload/Demo.app/Info.plist", data: infoPlist(t, "com.example.demo")},
				})
			},
		},
		{
			name: "duplicate ZIP entry",
			write: func(t *testing.T) string {
				plist := infoPlist(t, "com.example.demo")
				return writeExactErrorIPA(t, []exactErrorZIPEntry{
					{name: "Payload/Demo.app/Info.plist", data: plist},
					{name: "payload/demo.app/info.plist", data: plist},
				})
			},
		},
		{
			name: "missing embedded profile",
			write: func(t *testing.T) string {
				return writeExactErrorIPA(t, []exactErrorZIPEntry{
					{name: "Payload/Demo.app/Info.plist", data: infoPlist(t, "com.example.demo")},
				})
			},
		},
		{
			name: "invalid embedded profile",
			write: func(t *testing.T) string {
				return writeExactErrorIPA(t, []exactErrorZIPEntry{
					{name: "Payload/Demo.app/Info.plist", data: infoPlist(t, "com.example.demo")},
					{name: "Payload/Demo.app/embedded.mobileprovision", data: []byte("not CMS")},
				})
			},
		},
		{
			name: "invalid exact signing evidence",
			write: func(t *testing.T) string {
				return validIPA(t, []string{"registered-device"}, time.Now().Add(time.Hour), false)
			},
			configure: func(t *testing.T) {
				verifyCompleteSigningForTest = func(inspection *Inspection) {
					inspection.Signing.ProfileIntegrityVerification.Status = CodeSignatureVerified
					inspection.Signing.ProfileTrustVerification.Status = CodeSignatureVerified
					inspection.Signing.CodeSignatureVerification.Status = CodeSignatureInvalid
					inspection.Signing.CodeSignatureVerification.Scope = CodeSignatureScopeCompleteMainApp
				}
				t.Cleanup(func() { verifyCompleteSigningForTest = nil })
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.configure != nil {
				test.configure(t)
			}
			path := test.write(t)
			output := t.TempDir()
			_, err := prepareExactErrorPath(t, context.Background(), path, output)
			if !errors.Is(err, ErrNotEligible) {
				t.Fatalf("PrepareIPAPathExact() error = %v, want ErrNotEligible", err)
			}
			entries, readErr := os.ReadDir(output)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("deterministically invalid exact IPA wrote output: %#v", entries)
			}
		})
	}
}

func TestPrepareIPAPathExactKeepsToolUnavailableRetryable(t *testing.T) {
	path := validIPA(t, []string{"registered-device"}, time.Now().Add(time.Hour), false)
	verifyCompleteSigningForTest = func(inspection *Inspection) {
		inspection.Signing.ProfileIntegrityVerification.Status = CodeSignatureVerified
		inspection.Signing.ProfileTrustVerification.Status = CodeSignatureVerified
		inspection.Signing.CodeSignatureVerification.Status = CodeSignatureNotVerified
		inspection.Signing.CodeSignatureVerification.Scope = CodeSignatureScopeCompleteMainApp
	}
	t.Cleanup(func() { verifyCompleteSigningForTest = nil })

	_, err := prepareExactErrorPath(t, context.Background(), path, t.TempDir())
	if err == nil {
		t.Fatal("PrepareIPAPathExact() error = nil, want retryable verification failure")
	}
	if errors.Is(err, ErrNotEligible) {
		t.Fatalf("PrepareIPAPathExact() error = %v, tool unavailability must remain retryable", err)
	}
}

func TestCodeVerificationFailureClassifiesPermanentPathErrorsAsInvalid(t *testing.T) {
	permanent := &os.PathError{Op: "mkdir", Path: strings.Repeat("x", 256), Err: syscall.ENAMETOOLONG}
	status, _ := codeVerificationFailure(permanent, "main app")
	if status != CodeSignatureInvalid {
		t.Fatalf("ENAMETOOLONG status = %q, want %q", status, CodeSignatureInvalid)
	}

	transient := &os.PathError{Op: "write", Path: "Verify.app/Demo", Err: syscall.EIO}
	status, _ = codeVerificationFailure(transient, "main app")
	if status != CodeSignatureNotVerified {
		t.Fatalf("EIO status = %q, want %q", status, CodeSignatureNotVerified)
	}
}

func TestPrepareIPAPathExactKeepsCodeVerificationInfrastructureFailuresRetryable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign verification is macOS-only")
	}
	path := validExecutableIPA(t)
	trustExactErrorProfileForTest(t)

	tests := []struct {
		name      string
		injection func(*testing.T) (string, error)
	}{
		{
			name: "codesign fork-exec resource failure",
			injection: func(*testing.T) (string, error) {
				return "/usr/bin/codesign", &os.PathError{Op: "fork/exec", Path: "/usr/bin/codesign", Err: errors.New("resource temporarily unavailable")}
			},
		},
		{
			name: "lipo fork-exec resource failure",
			injection: func(*testing.T) (string, error) {
				return "/usr/bin/lipo", &os.PathError{Op: "fork/exec", Path: "/usr/bin/lipo", Err: errors.New("resource temporarily unavailable")}
			},
		},
		{
			name: "codesign signaled",
			injection: func(t *testing.T) (string, error) {
				t.Helper()
				err := exec.Command("/bin/sh", "-c", "kill -TERM $$").Run()
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) || exitErr.ProcessState == nil || exitErr.Exited() {
					t.Fatalf("signaled command error = %#v, want a signaled exec.ExitError", err)
				}
				return "/usr/bin/codesign", err
			},
		},
		{
			name: "codesign internal deadline",
			injection: func(*testing.T) (string, error) {
				return "/usr/bin/codesign", context.DeadlineExceeded
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			injectedTool, injectedErr := test.injection(t)
			runCodeSignTool = exactErrorCodeSignTool(t, injectedTool, injectedErr, false)
			t.Cleanup(func() { runCodeSignTool = runBoundedTool })

			_, err := prepareExactErrorPath(t, context.Background(), path, t.TempDir())
			if err == nil || errors.Is(err, ErrNotEligible) {
				t.Fatalf("PrepareIPAPathExact() error = %v, infrastructure failure must remain retryable", err)
			}
		})
	}
}

func TestPrepareIPAPathExactKeepsCertificateWorkspaceReadFailureRetryable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign verification is macOS-only")
	}
	path := validExecutableIPA(t)
	trustExactErrorProfileForTest(t)
	runCodeSignTool = exactErrorCodeSignTool(t, "", nil, true)
	t.Cleanup(func() { runCodeSignTool = runBoundedTool })

	_, err := prepareExactErrorPath(t, context.Background(), path, t.TempDir())
	if err == nil || errors.Is(err, ErrNotEligible) {
		t.Fatalf("PrepareIPAPathExact() error = %v, certificate workspace read failure must remain retryable", err)
	}
}

func TestPrepareIPAPathExactKeepsMaterializationWorkspaceIOFailureRetryable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign verification is macOS-only")
	}
	path := validExecutableIPA(t)
	trustExactErrorProfileForTest(t)
	originalMaterialize := materializeMainAppForVerification
	materializeMainAppForVerification = func(*os.Root, []*zip.File, string) error {
		return &os.PathError{Op: "write", Path: "Verify.app/Demo", Err: errors.New("input/output error")}
	}
	t.Cleanup(func() { materializeMainAppForVerification = originalMaterialize })

	_, err := prepareExactErrorPath(t, context.Background(), path, t.TempDir())
	if err == nil || errors.Is(err, ErrNotEligible) {
		t.Fatalf("PrepareIPAPathExact() error = %v, materialization workspace I/O must remain retryable", err)
	}
}

func TestPrepareIPAPathExactKeepsNormalCodeSignRejectionTerminal(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign verification is macOS-only")
	}
	path := validExecutableIPA(t)
	trustExactErrorProfileForTest(t)
	rejectedErr := exec.Command("/bin/sh", "-c", "exit 1").Run()
	var exitErr *exec.ExitError
	if !errors.As(rejectedErr, &exitErr) || exitErr.ProcessState == nil || !exitErr.Exited() {
		t.Fatalf("rejected command error = %#v, want a normal exec.ExitError", rejectedErr)
	}
	runCodeSignTool = exactErrorCodeSignTool(t, "/usr/bin/codesign", rejectedErr, false)
	t.Cleanup(func() { runCodeSignTool = runBoundedTool })

	_, err := prepareExactErrorPath(t, context.Background(), path, t.TempDir())
	if !errors.Is(err, ErrNotEligible) {
		t.Fatalf("PrepareIPAPathExact() error = %v, normal codesign rejection must be terminal", err)
	}
}

func TestPrepareIPAPathExactKeepsContextAndInputIOFailuresRetryable(t *testing.T) {
	inputDir := t.TempDir()
	root, err := rootfs.New(inputDir)
	if err != nil {
		t.Fatal(err)
	}
	expected := ExpectedIPA{SHA256: strings.Repeat("a", 64), SizeBytes: 1}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PrepareIPAPathExact(ctx, root, "missing.ipa", expected, PrepareOptions{Root: t.TempDir()}); !errors.Is(err, context.Canceled) || errors.Is(err, ErrNotEligible) {
		t.Fatalf("canceled PrepareIPAPathExact() error = %v", err)
	}
	if _, err := PrepareIPAPathExact(context.Background(), root, "missing.ipa", expected, PrepareOptions{Root: t.TempDir()}); err == nil || errors.Is(err, ErrNotEligible) {
		t.Fatalf("input I/O PrepareIPAPathExact() error = %v, want retryable non-eligibility-independent error", err)
	}
}

type exactErrorZIPEntry struct {
	name string
	data []byte
}

func writeExactErrorIPA(t *testing.T, entries []exactErrorZIPEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "Exact.ipa")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		member, createErr := writer.Create(entry.name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := member.Write(entry.data); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func prepareExactErrorPath(t *testing.T, ctx context.Context, path, output string) (PrepareResult, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	root, err := rootfs.New(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	return PrepareIPAPathExact(ctx, root, filepath.Base(path), ExpectedIPA{
		SHA256:    hex.EncodeToString(digest[:]),
		SizeBytes: int64(len(data)),
	}, PrepareOptions{Root: output})
}

func exactErrorCodeSignTool(t *testing.T, injectedTool string, injectedErr error, omitCertificate bool) func(context.Context, string, ...string) ([]byte, error) {
	t.Helper()
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == injectedTool {
			return nil, injectedErr
		}
		if name == "/usr/bin/lipo" {
			return []byte("arm64\n"), nil
		}
		if name != "/usr/bin/codesign" {
			t.Fatalf("unexpected verification tool %q", name)
		}
		for _, argument := range args {
			if argument == "--entitlements" {
				return plist.Marshal(map[string]any{
					"application-identifier":              "TEAM123.com.example.demo",
					"com.apple.developer.team-identifier": "TEAM123",
				}, plist.XMLFormat)
			}
		}
		for _, argument := range args {
			if strings.HasPrefix(argument, "--extract-certificates=") {
				if omitCertificate {
					return nil, nil
				}
				t.Fatalf("unexpected certificate extraction without injected failure")
			}
		}
		return nil, nil
	}
}

func trustExactErrorProfileForTest(t *testing.T) {
	t.Helper()
	verifyCompleteSigningForTest = func(inspection *Inspection) {
		inspection.Signing.ProfileIntegrityVerification.Status = CodeSignatureVerified
		inspection.Signing.ProfileTrustVerification.Status = CodeSignatureVerified
	}
	t.Cleanup(func() { verifyCompleteSigningForTest = nil })
}
