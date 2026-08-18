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
