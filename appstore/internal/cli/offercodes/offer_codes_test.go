package offercodes

import "testing"

func TestOfferCodesCommandConstructors(t *testing.T) {
	top := OfferCodesCommand()
	if top == nil {
		t.Fatal("expected offer-codes command")
	}
	if top.Name == "" {
		t.Fatal("expected command name")
	}
	if len(top.Subcommands) == 0 {
		t.Fatal("expected subcommands")
	}

	if got := OfferCodesCommand(); got == nil {
		t.Fatal("expected Command wrapper to return command")
	}

	constructors := []func() any{
		func() any { return OfferCodeCustomCodesCommand() },
		func() any { return OfferCodePricesCommand() },
		func() any { return OfferCodesGenerateCommand() },
	}
	for _, ctor := range constructors {
		if got := ctor(); got == nil {
			t.Fatal("expected constructor to return command")
		}
	}
}
