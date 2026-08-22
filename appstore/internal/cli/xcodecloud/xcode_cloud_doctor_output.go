package xcodecloud

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func printXcodeCloudDoctorResult(result *asc.XcodeCloudDoctorResult, output string, pretty bool) error {
	return shared.PrintOutput(result, output, pretty)
}
