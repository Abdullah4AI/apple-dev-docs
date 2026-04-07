package metadata

import (
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func effectiveVersionCreateLocalization(localPatch versionLocalPatch) VersionLocalization {
	createLoc := localPatch.localization
	if hasVersionContent(localPatch.createLocalization) {
		createLoc = localPatch.createLocalization
	}
	return NormalizeVersionLocalization(createLoc)
}

func versionCreateWarningForPatch(locale string, localPatch versionLocalPatch, remoteExists bool, mode string, opts shared.SubmitReadinessOptions) (shared.SubmitReadinessCreateWarning, bool) {
	if remoteExists {
		return shared.SubmitReadinessCreateWarning{}, false
	}
	return shared.SubmitReadinessCreateWarningForLocaleWithOptions(
		locale,
		versionAttributes(locale, effectiveVersionCreateLocalization(localPatch), true),
		mode,
		opts,
	)
}

func versionCreateWarningsForPatches(local map[string]versionLocalPatch, remote map[string]VersionLocalization, mode string, opts shared.SubmitReadinessOptions) []shared.SubmitReadinessCreateWarning {
	warnings := make([]shared.SubmitReadinessCreateWarning, 0, len(local))
	for _, locale := range sortedKeys(local) {
		warning, ok := versionCreateWarningForPatch(locale, local[locale], hasRemoteVersionLocalization(remote, locale), mode, opts)
		if !ok {
			continue
		}
		warnings = append(warnings, warning)
	}
	return shared.NormalizeSubmitReadinessCreateWarnings(warnings)
}

func versionCreateWarningsNeedUpdateContext(local map[string]versionLocalPatch, remote map[string]VersionLocalization) bool {
	for _, locale := range sortedKeys(local) {
		if hasRemoteVersionLocalization(remote, locale) {
			continue
		}
		attrs := versionAttributes(locale, effectiveVersionCreateLocalization(local[locale]), true)
		if strings.TrimSpace(attrs.WhatsNew) == "" {
			return true
		}
	}
	return false
}

func hasRemoteVersionLocalization(remote map[string]VersionLocalization, locale string) bool {
	if len(remote) == 0 {
		return false
	}
	_, ok := remote[locale]
	return ok
}
