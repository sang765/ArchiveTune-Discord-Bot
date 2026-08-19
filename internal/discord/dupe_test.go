package discord

import "testing"

func TestParseDupeCommand(t *testing.T) {
	link, matched, valid := ParseDupeCommand(".dupe https://discord.com/channels/123/456/789")
	if !matched || !valid || link != "https://discord.com/channels/123/456/789" {
		t.Fatalf("unexpected parse result: link=%q matched=%v valid=%v", link, matched, valid)
	}
	if _, matched, valid := ParseDupeCommand(".dupe"); !matched || valid {
		t.Fatalf("expected missing link to match command but be invalid")
	}
	if _, matched, _ := ParseDupeCommand(".duplicate https://discord.com/channels/123/456/789"); matched {
		t.Fatal("expected only .dupe to match this parser")
	}
}

func TestParseDiscordPostLink(t *testing.T) {
	guildID := "123456789012345678"
	postID, err := ParseDiscordPostLink("https://discord.com/channels/123456789012345678/987654321098765432/111111111111111111", guildID)
	if err != nil || postID != "987654321098765432" {
		t.Fatalf("unexpected post link parse: id=%q err=%v", postID, err)
	}
	postID, err = ParseDiscordPostLink("https://discord.com/channels/123456789012345678/987654321098765432", guildID)
	if err != nil || postID != "987654321098765432" {
		t.Fatalf("unexpected short post link parse: id=%q err=%v", postID, err)
	}
	if _, err := ParseDiscordPostLink("https://discord.com/channels/999999999999999999/987654321098765432/111111111111111111", guildID); err == nil {
		t.Fatal("expected different guild to be rejected")
	}
	if _, err := ParseDiscordPostLink("https://example.com/channels/123/456/789", guildID); err == nil {
		t.Fatal("expected non-Discord link to be rejected")
	}
}
