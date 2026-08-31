package asc

// XcodeToolchainDoctorResult is the computed output contract for the local
// Xcode toolchain diagnostic. It intentionally lives in the output package so
// JSON field names remain stable independently of the probe implementation.
type XcodeToolchainDoctorResult struct {
	Status       string                      `json:"status"`
	Source       string                      `json:"source,omitempty"`
	DeveloperDir string                      `json:"developerDir,omitempty"`
	XcodePath    string                      `json:"xcodePath,omitempty"`
	XcodeVersion string                      `json:"xcodeVersion,omitempty"`
	XcodeBuild   string                      `json:"xcodeBuild,omitempty"`
	Beta         *bool                       `json:"beta,omitempty"`
	Checks       []XcodeToolchainDoctorCheck `json:"checks"`
}

// XcodeToolchainDoctorCheck is one check in an XcodeToolchainDoctorResult.
type XcodeToolchainDoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

func xcodeToolchainDoctorResultRows(result *XcodeToolchainDoctorResult) ([]string, [][]string) {
	if result == nil {
		result = &XcodeToolchainDoctorResult{}
	}

	betaStatus := "unknown"
	betaMessage := "beta status unavailable because the selected developer directory was not inspected"
	if result.Beta != nil {
		if *result.Beta {
			betaStatus = "warn"
			betaMessage = "selected developer directory appears to be a beta Xcode build"
		} else {
			betaStatus = "ok"
			betaMessage = "selected developer directory is not identified as beta"
		}
	}

	rows := [][]string{
		{"source", "selected", "", result.Source},
		{"developer_dir", "selected", result.DeveloperDir, "effective developer directory"},
		{"xcode_path", "selected", result.XcodePath, "Xcode application, when identified"},
		{"xcode_version", "selected", result.XcodeVersion, result.XcodeBuild},
	}
	if !hasXcodeToolchainBetaCheck(result.Checks) {
		rows = append(rows, []string{"beta", betaStatus, "", betaMessage})
	}
	for _, check := range result.Checks {
		rows = append(rows, []string{check.Name, check.Status, check.Path, check.Message})
	}
	rows = append(rows, []string{"summary", result.Status, "", "overall toolchain status"})
	return []string{"check", "status", "path", "message"}, rows
}

func hasXcodeToolchainBetaCheck(checks []XcodeToolchainDoctorCheck) bool {
	for _, check := range checks {
		if check.Name == "beta" {
			return true
		}
	}
	return false
}
