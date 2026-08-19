package discord

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// ParseDupeCommand reports whether the message starts with .dupe and, if so,
// returns its single Discord link argument.
func ParseDupeCommand(content string) (link string, matched bool, valid bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) == 0 || !strings.EqualFold(fields[0], ".dupe") {
		return "", false, false
	}
	if len(fields) != 2 {
		return "", true, false
	}
	return fields[1], true, true
}

// ParseDiscordPostLink accepts a Discord channel/message link. For a reply
// message link, the channel segment is the forum post thread and is therefore
// the target post ID.
func ParseDiscordPostLink(rawLink, guildID string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawLink))
	if err != nil || parsed.Scheme != "https" {
		return "", fmt.Errorf("link phải là Discord HTTPS message link")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "discord.com" && host != "discordapp.com" && !strings.HasSuffix(host, ".discord.com") {
		return "", fmt.Errorf("link không thuộc Discord")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if (len(parts) != 3 && len(parts) != 4) || parts[0] != "channels" {
		return "", fmt.Errorf("định dạng link cần là https://discord.com/channels/<guild_id>/<post_id> hoặc .../<message_id>")
	}
	if parts[1] != guildID {
		return "", fmt.Errorf("link thuộc server khác")
	}
	if !isSnowflake(parts[2]) {
		return "", fmt.Errorf("link chứa post ID không hợp lệ")
	}
	if len(parts) == 4 && !isSnowflake(parts[3]) {
		return "", fmt.Errorf("link chứa message ID không hợp lệ")
	}
	return parts[2], nil
}

func isSnowflake(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (m *Manager) ApplyDuplicateAction(threadID string) (*discordgo.Channel, error) {
	return m.ApplySuggestionStatusAction(threadID, suggestionStatusActions[".dupe"])
}
