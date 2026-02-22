package shared

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseOptionalBoolFlag parses an optional boolean flag value.
func ParseOptionalBoolFlag(name, raw string) (*bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be true or false", name)
	}
	return &value, nil
}
