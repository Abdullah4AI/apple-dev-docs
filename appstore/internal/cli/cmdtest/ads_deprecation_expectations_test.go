package cmdtest

func adsV5Warning(oldPath, guidance string) string {
	return "Warning: `asc ads " + oldPath + "` is deprecated and retires on January 26, 2027. " + guidance + "\n"
}

func adsV5ReplacementWarning(oldPath, replacement string) string {
	return adsV5Warning(oldPath, "Use `asc ads "+replacement+"`.")
}

func adsV5NoReplacementWarning(oldPath, guidance string) string {
	return adsV5Warning(oldPath, guidance)
}
