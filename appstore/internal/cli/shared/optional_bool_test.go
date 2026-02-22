package shared

import (
	"flag"
	"io"
	"testing"
)

func TestOptionalBool_DefaultRequiresExplicitValue(t *testing.T) {
	var value OptionalBool
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&value, "flag", "test flag")

	if err := fs.Parse([]string{"--flag", "true"}); err != nil {
		t.Fatalf("expected parse to succeed with explicit value, got %v", err)
	}
	if !value.IsSet() || !value.Value() {
		t.Fatalf("expected flag to be set to true")
	}

	var missingValue OptionalBool
	fsMissing := flag.NewFlagSet("test", flag.ContinueOnError)
	fsMissing.SetOutput(io.Discard)
	fsMissing.Var(&missingValue, "flag", "test flag")
	err := fsMissing.Parse([]string{"--flag"})
	if err == nil {
		t.Fatal("expected parse error for missing flag value")
	}
}

func TestOptionalBool_EnableBoolFlagAllowsBareFlag(t *testing.T) {
	var value OptionalBool
	value.EnableBoolFlag()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&value, "flag", "test flag")

	if !value.IsBoolFlag() {
		t.Fatal("expected bool flag mode to be enabled")
	}
	if err := fs.Parse([]string{"--flag"}); err != nil {
		t.Fatalf("expected bare bool flag parse to succeed, got %v", err)
	}
	if !value.IsSet() || !value.Value() {
		t.Fatalf("expected bare flag to set value=true")
	}
}

func TestOptionalBool_EnableBoolFlagSupportsExplicitFalse(t *testing.T) {
	var value OptionalBool
	value.EnableBoolFlag()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&value, "flag", "test flag")

	if err := fs.Parse([]string{"--flag=false"}); err != nil {
		t.Fatalf("expected explicit false parse to succeed, got %v", err)
	}
	if !value.IsSet() {
		t.Fatal("expected flag to be marked set")
	}
	if value.Value() {
		t.Fatal("expected flag value=false")
	}
}
