package cmdtest

import "testing"

func TestWebAPIKeysCreateCommandRegistration(t *testing.T) {
	root := RootCommand("1.2.3")
	cmd := findSubcommand(root, "web", "api-keys", "create")
	if cmd == nil {
		t.Fatal("expected web api-keys create command")
	}
	for _, flagName := range []string{
		"name",
		"role",
		"output-dir",
		"apple-id",
		"two-factor-code-command",
		"provider-id",
		"public-provider-id",
		"output",
		"pretty",
	} {
		if cmd.FlagSet.Lookup(flagName) == nil {
			t.Fatalf("expected --%s flag", flagName)
		}
	}
	if got := cmd.FlagSet.Lookup("role").DefValue; got != "ADMIN" {
		t.Fatalf("expected --role default ADMIN, got %q", got)
	}
}
