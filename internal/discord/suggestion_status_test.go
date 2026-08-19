package discord

import "testing"

func TestParseSuggestionStatusCommand(t *testing.T) {
	cases := []struct {
		content string
		command string
		tag     string
		prefix  string
		link    string
	}{
		{content: ".dupe https://discord.com/channels/123/456/789", command: ".dupe", tag: "Duplicate", prefix: "[DUPLICATE]", link: "https://discord.com/channels/123/456/789"},
		{content: ".done", command: ".done", tag: "Done", prefix: "[DONE]"},
		{content: ".in-progress", command: ".in-progress", tag: "In Progress...", prefix: "[IN PROGRESS]"},
		{content: ".exist", command: ".exist", tag: "Already exist", prefix: "[ALREADY EXIST]"},
	}
	for _, testCase := range cases {
		action, link, matched, valid := ParseSuggestionStatusCommand(testCase.content)
		if !matched || !valid || link != testCase.link {
			t.Fatalf("unexpected parse result for %q: action=%#v link=%q matched=%v valid=%v", testCase.content, action, link, matched, valid)
		}
		if action.Command != testCase.command || action.TagName != testCase.tag || action.TitlePrefix != testCase.prefix {
			t.Fatalf("unexpected action for %q: %#v", testCase.content, action)
		}
	}
}

func TestParseSuggestionStatusCommandRejectsUnexpectedArguments(t *testing.T) {
	if _, _, matched, valid := ParseSuggestionStatusCommand(".dupe"); !matched || valid {
		t.Fatal("expected .dupe without link to be matched but invalid")
	}
	if _, _, matched, valid := ParseSuggestionStatusCommand(".exist https://discord.com/channels/123/456"); !matched || valid {
		t.Fatal("expected .exist with an extra link to be invalid")
	}
	if _, _, matched, _ := ParseSuggestionStatusCommand(".maybe https://discord.com/channels/123/456"); matched {
		t.Fatal("expected .maybe not to be a linked suggestion status command")
	}
}
