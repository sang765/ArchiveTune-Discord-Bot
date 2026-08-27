package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BotToken            string          `yaml:"bot_token"`
	GuildID             string          `yaml:"guild_id"`
	ModeratorRoleIDs    []string        `yaml:"moderator_role_ids"`
	ReplaceExistingTags bool            `yaml:"replace_existing_tags"`
	PrefixAutocorrect   bool            `yaml:"prefix_autocorrect"`
	PrefixMaxDistance   int             `yaml:"prefix_max_distance"`
	YTD                 YTDConfig       `yaml:"ytd"`
	Channels            []ChannelConfig `yaml:"channels"`
}

type YTDConfig struct {
	BlockPlaylistAlbumDownload *bool `yaml:"block_playlist_album_download"`
}

func (c YTDConfig) BlockPlaylistAlbumDownloadEnabled() bool {
	return c.BlockPlaylistAlbumDownload == nil || *c.BlockPlaylistAlbumDownload
}

type ChannelConfig struct {
	ID             string      `yaml:"id"`
	Name           string      `yaml:"name"`
	RequireTag     bool        `yaml:"require_tag"`
	PostGuidelines string      `yaml:"post_guidelines"`
	Tags           []TagConfig `yaml:"tags"`
}

type TagConfig struct {
	Name          string `yaml:"name"`
	Emoji         string `yaml:"emoji"`
	EmojiID       string `yaml:"emoji_id"`
	EmojiAnimated bool   `yaml:"emoji_animated"`
}

var customEmojiPattern = regexp.MustCompile(`^<(a?):([A-Za-z0-9_~]+):([0-9]{18,20})>$`)

// DiscordEmoji returns the ForumTag emoji fields. The emoji field accepts a
// Unicode emoji or a Discord custom emoji formatted as <:name:id> / <a:name:id>.
func (t TagConfig) DiscordEmoji() (emojiName, emojiID string, animated bool, err error) {
	raw := strings.TrimSpace(t.Emoji)
	if match := customEmojiPattern.FindStringSubmatch(raw); match != nil {
		parsedAnimated := match[1] == "a"
		if t.EmojiID != "" && t.EmojiID != match[3] {
			return "", "", false, fmt.Errorf("tag %q has conflicting custom emoji ids", t.Name)
		}
		if t.EmojiAnimated && !parsedAnimated {
			return "", "", false, fmt.Errorf("tag %q marks a static custom emoji as animated", t.Name)
		}
		// Discord Forum Tags require emoji_id for custom emoji and reject a
		// payload that also contains emoji_name. The name is only needed in
		// message markup, not in the ForumTag API payload.
		return "", match[3], parsedAnimated, nil
	}
	if strings.HasPrefix(raw, "<") || strings.Contains(raw, ":") && strings.HasSuffix(raw, ">") {
		return "", "", false, fmt.Errorf("tag %q has an invalid custom emoji; use <:name:id> or <a:name:id>", t.Name)
	}
	if t.EmojiID != "" {
		if !isEmojiID(t.EmojiID) {
			return "", "", false, fmt.Errorf("tag %q has an invalid emoji_id", t.Name)
		}
		return "", t.EmojiID, t.EmojiAnimated, nil
	}
	if t.EmojiAnimated {
		return "", "", false, fmt.Errorf("tag %q sets emoji_animated without emoji_id or a custom emoji value", t.Name)
	}
	return raw, "", false, nil
}

func isEmojiID(value string) bool {
	if len(value) < 18 || len(value) > 20 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
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
	if cfg.PrefixMaxDistance == 0 {
		cfg.PrefixMaxDistance = 2
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
	if c.PrefixMaxDistance < 0 || c.PrefixMaxDistance > 3 {
		return errors.New("prefix_max_distance must be between 0 and 3")
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
			if _, _, _, err := tag.DiscordEmoji(); err != nil {
				return err
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
