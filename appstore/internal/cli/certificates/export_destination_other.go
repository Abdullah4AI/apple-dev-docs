//go:build !darwin

package certificates

func certificateExportRootPath(output string) string {
	return output
}
