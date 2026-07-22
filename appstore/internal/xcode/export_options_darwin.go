//go:build darwin

package xcode

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	legacyexportoptions "github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/exportoptionsgenerator"
	"github.com/bitrise-io/go-xcode/v2/xcarchive"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
)

var bitriseStdoutCaptureMu sync.Mutex

// buildPlatformExportOptionsPayload uses Bitrise's current typed v2 model on
// macOS, where xcodebuild and local signing asset resolution are available.
func buildPlatformExportOptionsPayload(opts ExportOptionsGenerateOptions, teamID string, manual manualExportOptions) map[string]any {
	model := exportoptions.NewAppStoreConnectOptions(exportoptions.MethodAppStoreConnect)
	model.TeamID = teamID
	model.Destination = exportoptions.Destination(opts.Destination)
	model.SigningStyle = exportoptions.SigningStyle(opts.SigningStyle)
	if opts.SigningStyle == exportOptionsSigningStyleManual {
		model.SigningCertificate = manual.SigningCertificate
		model.BundleIDProvisioningProfileMapping = cloneProvisioningProfiles(manual.ProvisioningProfiles)
		model.ICloudContainerEnvironment = exportoptions.ICloudContainerEnvironment(manual.ICloudContainerEnvironment)
	}
	return model.Hash()
}

func generateManualExportOptions(ctx context.Context, archivePath, teamID string) (manualExportOptions, error) {
	if err := contextError(ctx); err != nil {
		return manualExportOptions{}, err
	}
	platform, err := InferArchivePlatform(archivePath)
	if err != nil {
		return manualExportOptions{}, fmt.Errorf("infer archive platform: %w", err)
	}
	if platform != "IOS" && platform != "TV_OS" {
		return manualExportOptions{}, fmt.Errorf("manual signing export options generation only supports iOS and tvOS archives; archive platform is %s", platform)
	}
	archive, err := xcarchive.NewIosArchive(archivePath)
	if err != nil {
		return manualExportOptions{}, fmt.Errorf("read iOS archive: %w", err)
	}
	archiveInfo, err := exportoptionsgenerator.ReadArchiveExportInfo(archive)
	if err != nil {
		return manualExportOptions{}, fmt.Errorf("read archive export information: %w", err)
	}
	var generated legacyexportoptions.ExportOptions
	if _, err := captureBitriseStdout(func() error {
		generator := exportoptionsgenerator.New(
			xcodeversion.NewXcodeVersionProvider(command.NewFactory(env.NewRepository())),
			log.NewLogger(),
		)
		var generateErr error
		generated, generateErr = generator.GenerateApplicationExportOptions(
			exportoptionsgenerator.ExportProductApp,
			archiveInfo,
			// Bitrise v2's generator currently exposes these v1 argument types.
			legacyexportoptions.MethodAppStoreConnect,
			legacyexportoptions.SigningStyleManual,
			exportoptionsgenerator.Opts{TeamID: teamID},
		)
		return generateErr
	}); err != nil {
		return manualExportOptions{}, err
	}
	return manualExportOptionsFromHash(generated.Hash())
}

// captureBitriseStdout contains upstream status prints so structured CLI
// output remains valid. Bitrise does not currently expose a writer for these
// messages, and os.Stdout is process-global, so captures are serialized.
func captureBitriseStdout(run func() error) (string, error) {
	bitriseStdoutCaptureMu.Lock()
	defer bitriseStdoutCaptureMu.Unlock()

	reader, writer, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("capture Bitrise stdout: %w", err)
	}
	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
		_ = writer.Close()
		_ = reader.Close()
	}()

	type readResult struct {
		data []byte
		err  error
	}
	readDone := make(chan readResult, 1)
	go func() {
		data, readErr := io.ReadAll(reader)
		readDone <- readResult{data: data, err: readErr}
	}()

	os.Stdout = writer
	runErr := run()
	os.Stdout = originalStdout
	closeErr := writer.Close()
	result := <-readDone

	if runErr != nil {
		return string(result.data), runErr
	}
	if closeErr != nil {
		return string(result.data), fmt.Errorf("close Bitrise stdout capture: %w", closeErr)
	}
	if result.err != nil {
		return string(result.data), fmt.Errorf("read Bitrise stdout capture: %w", result.err)
	}
	return string(result.data), nil
}

func manualExportOptionsFromHash(payload map[string]interface{}) (manualExportOptions, error) {
	profiles, err := provisioningProfilesFromPayload(payload["provisioningProfiles"])
	if err != nil {
		return manualExportOptions{}, err
	}
	cloudEnvironment := ""
	if value, ok := payload["iCloudContainerEnvironment"]; ok {
		cloudEnvironment = strings.TrimSpace(fmt.Sprint(value))
	}
	return manualExportOptions{
		TeamID:                     strings.TrimSpace(coercePlistValueToString(payload["teamID"])),
		SigningCertificate:         strings.TrimSpace(coercePlistValueToString(payload["signingCertificate"])),
		ProvisioningProfiles:       profiles,
		ICloudContainerEnvironment: cloudEnvironment,
	}, nil
}
