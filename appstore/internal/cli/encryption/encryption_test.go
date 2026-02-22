package encryption

import "testing"

func TestEncryptionCommandConstructors(t *testing.T) {
	top := EncryptionCommand()
	if top == nil {
		t.Fatal("expected encryption command")
	}
	if top.Name == "" {
		t.Fatal("expected command name")
	}
	if len(top.Subcommands) == 0 {
		t.Fatal("expected subcommands")
	}

	if got := EncryptionCommand(); got == nil {
		t.Fatal("expected Command wrapper to return command")
	}

	constructors := []func() any{
		func() any { return EncryptionDeclarationsCommand() },
		func() any { return EncryptionDocumentsCommand() },
		func() any { return EncryptionDeclarationsAppCommand() },
	}
	for _, ctor := range constructors {
		if got := ctor(); got == nil {
			t.Fatal("expected constructor to return command")
		}
	}
}
