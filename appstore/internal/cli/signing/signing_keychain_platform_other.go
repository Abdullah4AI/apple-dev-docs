//go:build !darwin

package signing

func platformSigningKeychainInstallDeps() signingKeychainInstallDeps {
	return signingKeychainInstallDeps{GOOS: "unsupported"}
}
