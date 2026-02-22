package alternativedistribution

import (
	"fmt"
	"os"
	"strings"
)

const alternativeDistributionMaxLimit = 200

func readPublicKey(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("public key path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("public key file is empty")
	}
	return value, nil
}
