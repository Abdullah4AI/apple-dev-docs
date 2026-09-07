package cmdtest

import "testing"

func TestWebCustomUserRecipientCommandsRegistered(t *testing.T) {
	root := RootCommand("1.2.3")
	for _, leaf := range []string{"list", "create", "delete"} {
		path := []string{"web", "apps", "distribution", "users", leaf}
		if command := findSubcommand(root, path...); command == nil {
			t.Fatalf("expected %v to be registered", path)
		}
	}
}
