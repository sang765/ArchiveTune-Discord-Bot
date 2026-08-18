package discord

import "testing"

func TestParsePrefixCommandSolvedAndAlias(t *testing.T) {
	for _, command := range []string{".solved", ".sloved"} {
		action, ok := ParsePrefixCommand(command)
		if !ok {
			t.Fatalf("expected %s to be recognized", command)
		}
		if action.TagName != "Solved" || action.TitlePrefix != "[SOLVED]" {
			t.Fatalf("unexpected action for %s: %#v", command, action)
		}
	}
}

func TestParsePrefixCommandRejectsArguments(t *testing.T) {
	if _, ok := ParsePrefixCommand(".solved extra"); ok {
		t.Fatal("expected command with arguments to be rejected")
	}
}
