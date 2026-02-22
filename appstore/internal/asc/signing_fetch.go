package asc

// SigningFetchResult represents CLI output for signing fetch.
type SigningFetchResult struct {
	BundleID         string   `json:"bundleId"`
	BundleIDResource string   `json:"bundleIdResourceId"`
	ProfileType      string   `json:"profileType"`
	ProfileID        string   `json:"profileId"`
	ProfileFile      string   `json:"profileFile"`
	CertificateIDs   []string `json:"certificateIds"`
	CertificateFiles []string `json:"certificateFiles"`
	OutputPath       string   `json:"outputPath"`
	Created          bool     `json:"created,omitempty"`
}
