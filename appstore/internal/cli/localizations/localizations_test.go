package localizations

import "testing"

func TestLocalizationsCommandConstructors(t *testing.T) {
	top := LocalizationsCommand()
	if top == nil {
		t.Fatal("expected localizations command")
	}
	if top.Name == "" {
		t.Fatal("expected command name")
	}
	if len(top.Subcommands) == 0 {
		t.Fatal("expected subcommands")
	}

	if got := LocalizationsCommand(); got == nil {
		t.Fatal("expected Command wrapper to return command")
	}

	if got := LocalizationsPreviewSetsCommand(); got == nil {
		t.Fatal("expected preview sets command")
	}
	if got := LocalizationsSearchKeywordsCommand(); got == nil {
		t.Fatal("expected search keywords command")
	}
}
