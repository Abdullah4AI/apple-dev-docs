package shared

import (
	"fmt"
	"strconv"
)

// OptionalBool tracks whether a boolean flag was explicitly set.
type OptionalBool struct {
	set   bool
	value bool
	// boolFlag controls whether flag.Parse can accept bare --flag syntax.
	// When false (default), an explicit value is required (e.g. --flag true).
	boolFlag bool
}

func (b *OptionalBool) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("must be true or false")
	}
	b.value = parsed
	b.set = true
	return nil
}

func (b *OptionalBool) String() string {
	if !b.set {
		return ""
	}
	return strconv.FormatBool(b.value)
}

// IsBoolFlag tells the standard flag parser whether bare --flag is allowed.
func (b *OptionalBool) IsBoolFlag() bool {
	return b.boolFlag
}

// EnableBoolFlag allows bare --flag syntax for this OptionalBool instance.
func (b *OptionalBool) EnableBoolFlag() {
	b.boolFlag = true
}

func (b OptionalBool) IsSet() bool {
	return b.set
}

func (b OptionalBool) Value() bool {
	return b.value
}
