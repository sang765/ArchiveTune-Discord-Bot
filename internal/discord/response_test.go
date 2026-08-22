package discord

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestResponseEmbeds(t *testing.T) {
	cases := []struct {
		name  string
		embed *discordgo.MessageEmbed
		color int
	}{
		{name: "info", embed: InfoEmbed("Usage", "details"), color: ResponseColorInfo},
		{name: "success", embed: SuccessEmbed("Done", "details"), color: ResponseColorSuccess},
		{name: "error", embed: ErrorEmbed("Failed", "details"), color: ResponseColorError},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if !strings.HasPrefix(testCase.embed.Title, "ArchiveTune Bot") {
				t.Fatalf("expected ArchiveTune Bot title, got %q", testCase.embed.Title)
			}
			if testCase.embed.Color != testCase.color {
				t.Fatalf("expected color %d, got %d", testCase.color, testCase.embed.Color)
			}
		})
	}
}

func TestWorkflowEmbedUsesBotAndGuildMetadata(t *testing.T) {
	state := discordgo.NewState()
	state.User = &discordgo.User{ID: "bot-1", Username: "ArchiveTune Bot", Avatar: "bot-avatar"}
	if err := state.GuildAdd(&discordgo.Guild{ID: "guild-1", Name: "ArchiveTune Community", Icon: "guild-icon"}); err != nil {
		t.Fatalf("add guild to state: %v", err)
	}

	session := &discordgo.Session{State: state}
	embed := WorkflowEmbed(session, "guild-1", "✅ Your issue has been marked as `solved`!!!", "details", ResponseColorSuccess)
	if embed.Author == nil || embed.Author.Name != "ArchiveTune Bot" {
		t.Fatalf("unexpected embed author: %#v", embed.Author)
	}
	if embed.Author.IconURL == "" {
		t.Fatal("expected bot avatar URL")
	}
	if embed.Footer == nil || embed.Footer.Text != "ArchiveTune Community" {
		t.Fatalf("unexpected embed footer: %#v", embed.Footer)
	}
	if embed.Footer.IconURL == "" {
		t.Fatal("expected guild icon URL")
	}
}
