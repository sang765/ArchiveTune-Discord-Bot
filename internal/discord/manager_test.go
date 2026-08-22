package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/config"
)

func TestMergeTagsPreservesIDsAndUpdatesDeclaredTags(t *testing.T) {
	existing := []discordgo.ForumTag{
		{ID: "1", Name: "Problem", EmojiName: "old"},
		{ID: "2", Name: "Legacy"},
	}
	declared := []config.TagConfig{{Name: "Problem", Emoji: "❗"}, {Name: "Question", Emoji: "❓"}}

	got, err := mergeTags(existing, declared, false)
	if err != nil {
		t.Fatalf("merge tags: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(got))
	}
	if got[0].ID != "1" || got[0].EmojiName != "❗" {
		t.Fatalf("expected existing tag ID and new emoji, got %#v", got[0])
	}
	if got[2].Name != "Question" {
		t.Fatalf("expected appended Question tag, got %#v", got[2])
	}
}

func TestMergeTagsCanReplaceExistingTags(t *testing.T) {
	got, err := mergeTags([]discordgo.ForumTag{{ID: "1", Name: "Legacy"}}, []config.TagConfig{{Name: "Problem"}}, true)
	if err != nil {
		t.Fatalf("merge tags: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Problem" {
		t.Fatalf("expected only declared tag, got %#v", got)
	}
}

func TestMergeTagsUsesCustomEmojiID(t *testing.T) {
	got, err := mergeTags(nil, []config.TagConfig{{Name: "Version", Emoji: "<:02V:1520005999040266240>"}}, false)
	if err != nil {
		t.Fatalf("merge custom emoji tag: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one tag, got %d", len(got))
	}
	if got[0].EmojiID != "1520005999040266240" || got[0].EmojiName != "02V" {
		t.Fatalf("unexpected custom emoji fields: %#v", got[0])
	}
}
