package shared

import (
	"flag"
	"fmt"
)

// OnceCSVValue is a flag.Value for comma-separated list flags that rejects
// repeated flag occurrences instead of silently keeping only the last value.
type OnceCSVValue struct {
	flagName string
	value    string
	set      bool
}

// BindOnceCSVFlag registers a comma-separated list flag that errors when the
// flag is passed more than once.
func BindOnceCSVFlag(fs *flag.FlagSet, name, usage string) *OnceCSVValue {
	value := &OnceCSVValue{flagName: name}
	fs.Var(value, name, usage)
	return value
}

func (v *OnceCSVValue) String() string { return v.value }

// Provided reports whether the flag was set at least once, including values
// recovered after a space-separated boolean flag.
func (v *OnceCSVValue) Provided() bool {
	return v != nil && v.set
}

func (v *OnceCSVValue) Set(raw string) error {
	if v.set {
		return fmt.Errorf(
			"--%s specified multiple times; pass one comma-separated list, for example --%s %q",
			v.flagName, v.flagName, v.value+","+raw,
		)
	}
	v.value = raw
	v.set = true
	return nil
}
