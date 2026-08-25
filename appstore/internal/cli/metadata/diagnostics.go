package metadata

import (
	"fmt"
	"os"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func metadataRequiredInputError(parameter, message string) error {
	trimmed := strings.TrimSpace(message)
	fmt.Fprintf(os.Stderr, "Error: %s\n", trimmed)
	return shared.WithDiagnostic(
		shared.NewReportedUsageError(shared.UsageErrorMissingRequired, trimmed),
		shared.DiagnosticRequiredInputMissing,
		parameter,
	)
}
