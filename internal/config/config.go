package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BotToken            string          `yaml:"bot_token"`
	GuildID             string          `yaml:"guild_id"`
	ModeratorRoleIDs    []string        `yaml:"moderator_role_ids"`
	ReplaceExistingTags bool            `yaml:"replace_existing_tags"`
	Channels            []ChannelConfig `yaml:"channels"`
}

type ChannelConfig struct {
	ID             string      `yaml:"id"`
	Name           string      `yaml:"name"`
	RequireTag     bool        `yaml:"require_tag"`
	PostGuidelines string      `yaml:"post_guidelines"`
	Tags           []TagConfig `yaml:"tags"`
}

type TagConfig struct {
	Name  string `yaml:"name"`
	Emoji string `yaml:"emoji"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.BotToken) == "" || strings.Contains(c.BotToken, "replace-with") {
		return errors.New("bot_token is missing")
	}
	if strings.TrimSpace(c.GuildID) == "" || strings.Contains(c.GuildID, "replace-with") {
		return errors.New("guild_id is missing")
	}
	if len(c.Channels) == 0 {
		return errors.New("at least one managed channel is required")
	}

	seenChannels := make(map[string]struct{}, len(c.Channels))
	for _, channel := range c.Channels {
		if strings.TrimSpace(channel.ID) == "" || strings.Contains(channel.ID, "replace-with") {
			return fmt.Errorf("channel %q has an invalid id", channel.Name)
		}
		if _, ok := seenChannels[channel.ID]; ok {
			return fmt.Errorf("duplicate channel id %q", channel.ID)
		}
		seenChannels[channel.ID] = struct{}{}
		if strings.TrimSpace(channel.Name) == "" {
			return fmt.Errorf("channel %q has an empty name", channel.ID)
		}
		seenTags := make(map[string]struct{}, len(channel.Tags))
		for _, tag := range channel.Tags {
			name := strings.TrimSpace(tag.Name)
			if name == "" {
				return fmt.Errorf("channel %q contains an empty tag name", channel.Name)
			}
			key := strings.ToLower(name)
			if _, ok := seenTags[key]; ok {
				return fmt.Errorf("channel %q contains duplicate tag %q", channel.Name, name)
			}
			seenTags[key] = struct{}{}
		}
	}
	return nil
}

func (c *Config) Channel(id string) (ChannelConfig, bool) {
	for _, channel := range c.Channels {
		if channel.ID == id {
			return channel, true
		}
	}
	return ChannelConfig{}, false
}
