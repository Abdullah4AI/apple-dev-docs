package asc

import "fmt"

// SigningResignInputResult identifies the source artifact without exposing
// its local path.
type SigningResignInputResult struct {
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}

// SigningResignArtifactResult identifies a generated artifact and its digest.
type SigningResignArtifactResult struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}

// SigningResignIdentityResult reports the certificate binding used for the
// local signing operation.
type SigningResignIdentityResult struct {
	CertificateSHA256 string `json:"certificateSha256"`
	TeamID            string `json:"teamId"`
}

// SigningResignTargetResult reports one validated app-like target.
type SigningResignTargetResult struct {
	Kind          string `json:"kind"`
	RelativePath  string `json:"relativePath"`
	BundleID      string `json:"bundleId"`
	ProfileClass  string `json:"profileClass"`
	ProfileUUID   string `json:"profileUuid"`
	ProfileSHA256 string `json:"profileSha256"`
	Status        string `json:"status"`
}

// SigningResignVerification reports the scope and result of post-signing
// verification.
type SigningResignVerification struct {
	Status string `json:"status"`
	Scope  string `json:"scope"`
}

// SigningResignResult is the stable computed output contract for
// `asc signing resign`.
type SigningResignResult struct {
	SchemaVersion int                         `json:"schemaVersion"`
	Command       string                      `json:"command"`
	Input         SigningResignInputResult    `json:"input"`
	Output        SigningResignArtifactResult `json:"output"`
	Identity      SigningResignIdentityResult `json:"identity"`
	Targets       []SigningResignTargetResult `json:"targets"`
	Verification  SigningResignVerification   `json:"verification"`
}

func signingResignResultRows(result *SigningResignResult) ([]string, [][]string) {
	rows := [][]string{
		{"schemaVersion", fmt.Sprintf("%d", result.SchemaVersion)},
		{"command", result.Command},
		{"input.sizeBytes", fmt.Sprintf("%d", result.Input.SizeBytes)},
		{"input.sha256", result.Input.SHA256},
		{"output.path", result.Output.Path},
		{"output.sizeBytes", fmt.Sprintf("%d", result.Output.SizeBytes)},
		{"output.sha256", result.Output.SHA256},
		{"identity.certificateSha256", result.Identity.CertificateSHA256},
		{"identity.teamId", result.Identity.TeamID},
		{"verification.status", result.Verification.Status},
		{"verification.scope", result.Verification.Scope},
	}
	for _, target := range result.Targets {
		prefix := "target." + target.RelativePath
		rows = append(
			rows,
			[]string{prefix + ".kind", target.Kind},
			[]string{prefix + ".bundleId", target.BundleID},
			[]string{prefix + ".profileClass", target.ProfileClass},
			[]string{prefix + ".profileUuid", target.ProfileUUID},
			[]string{prefix + ".profileSha256", target.ProfileSHA256},
			[]string{prefix + ".status", target.Status},
		)
	}
	return []string{"field", "value"}, rows
}
