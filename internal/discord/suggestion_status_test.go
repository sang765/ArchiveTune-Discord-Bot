package discord

import "testing"

func TestParseSuggestionStatusCommand(t *testing.T) {
	cases := []struct {
		content string
		command string
		tag     string
		prefix  string
	}{
		{content: ".dupe https://discord.com/channels/123/456/789", command: ".dupe", tag: "Duplicate", prefix: "[DUPLICATE]"},
		{content: ".done https://discord.com/channels/123/456", command: ".done", tag: "Done", prefix: "[DONE]"},
		{content: ".in-progress https://discord.com/channels/123/456", command: ".in-progress", tag: "In Progress...", prefix: "[IN PROGRESS]"},
		{content: ".exist https://discord.com/channels/123/456", command: ".exist", tag: "Already exist", prefix: "[ALREADY EXIST]"},
	}
	for _, testCase := range cases {
		action, link, matched, valid := ParseSuggestionStatusCommand(testCase.content)
		if !matched || !valid || link == "" {
			t.Fatalf("expected valid parse for %q", testCase.content)
		}
		if action.Command != testCase.command || action.TagName != testCase.tag || action.TitlePrefix != testCase.prefix {
			t.Fatalf("unexpected action for %q: %#v", testCase.content, action)
		}
	}
}

func TestParseSuggestionStatusCommandRequiresLink(t *testing.T) {
	if _, _, matched, valid := ParseSuggestionStatusCommand(".exist"); !matched || valid {
		t.Fatal("expected .exist without link to be matched but invalid")
	}
	if _, _, matched, _ := ParseSuggestionStatusCommand(".maybe https://discord.com/channels/123/456"); matched {
		t.Fatal("expected .maybe not to be a linked suggestion status command")
	}
}
