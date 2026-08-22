package config

import "testing"

func TestValidateRejectsPlaceholderAndDuplicateTags(t *testing.T) {
	cfg := Config{
		BotToken: "token",
		GuildID:  "guild",
		Channels: []ChannelConfig{{
			ID:   "channel",
			Name: "issues",
			Tags: []TagConfig{{Name: "Problem"}, {Name: "problem"}},
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate tag validation error")
	}
}

func TestValidateAcceptsConfiguredChannels(t *testing.T) {
	cfg := Config{
		BotToken: "token",
		GuildID:  "guild",
		Channels: []ChannelConfig{{ID: "channel", Name: "issues", Tags: []TagConfig{{Name: "Problem"}}}},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestTagConfigParsesCustomEmoji(t *testing.T) {
	name, id, animated, err := (TagConfig{Name: "Version", Emoji: "<:02V:1520005999040266240>"}).DiscordEmoji()
	if err != nil {
		t.Fatalf("parse custom emoji: %v", err)
	}
	if name != "02V" || id != "1520005999040266240" || animated {
		t.Fatalf("unexpected static custom emoji: name=%q id=%q animated=%v", name, id, animated)
	}

	name, id, animated, err = (TagConfig{Name: "Animated", Emoji: "<a:02V:1520005999040266240>"}).DiscordEmoji()
	if err != nil {
		t.Fatalf("parse animated custom emoji: %v", err)
	}
	if name != "02V" || id != "1520005999040266240" || !animated {
		t.Fatalf("unexpected animated custom emoji: name=%q id=%q animated=%v", name, id, animated)
	}
}

func TestTagConfigRejectsMalformedCustomEmoji(t *testing.T) {
	if _, _, _, err := (TagConfig{Name: "Broken", Emoji: "<:02V:not-an-id>"}).DiscordEmoji(); err == nil {
		t.Fatal("expected malformed custom emoji error")
	}
}
