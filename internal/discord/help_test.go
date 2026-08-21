package discord

import (
	"strings"
	"testing"
)

func TestHelpEmbedListsCommandGroups(t *testing.T) {
	embed := HelpEmbed()
	if embed == nil {
		t.Fatal("expected help embed")
	}
	if embed.Title != "ArchiveTune Bot Help" {
		t.Fatalf("unexpected title: %q", embed.Title)
	}

	content := embed.Title + "\n" + embed.Description
	for _, field := range embed.Fields {
		content += "\n" + field.Name + "\n" + field.Value
	}
	for _, command := range []string{".help", "/help", ".solved", ".accept", ".dupe <post link>", ".reject <reason>", "/forum-sync", "/fix-suggestion", "/post-state"} {
		if !strings.Contains(content, command) {
			t.Errorf("help embed does not contain %q", command)
		}
	}
}
