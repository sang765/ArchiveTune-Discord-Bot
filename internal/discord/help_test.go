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
	for _, command := range []string{"/help", "/solved", "/accept", "/dupe <post_link>", "/reject <reason>", "/forum-sync", "/fix-suggestion", "/post-state"} {
		if !strings.Contains(content, command) {
			t.Errorf("help embed does not contain %q", command)
		}
	}
}

func TestHelpEmbedHasNoPrefixCommands(t *testing.T) {
	embed := HelpEmbed()
	for _, field := range embed.Fields {
		if strings.Contains(field.Value, "`.solved`") || strings.Contains(field.Value, "`.ytd") {
			t.Fatalf("help embed still advertises prefix commands: %s", field.Value)
		}
	}
	if !strings.Contains(embed.Footer.Text, "Slash commands only") {
		t.Fatalf("unexpected footer: %q", embed.Footer.Text)
	}
}
