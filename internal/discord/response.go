package discord

import "github.com/bwmarrin/discordgo"

const (
	ResponseColorInfo    = 0x5865F2
	ResponseColorSuccess = 0x57F287
	ResponseColorError   = 0xED4245
)

func InfoEmbed(title, description string) *discordgo.MessageEmbed {
	return responseEmbed("ArchiveTune Bot · "+title, description, ResponseColorInfo)
}

func SuccessEmbed(title, description string) *discordgo.MessageEmbed {
	return responseEmbed("ArchiveTune Bot · "+title, description, ResponseColorSuccess)
}

func ErrorEmbed(title, description string) *discordgo.MessageEmbed {
	return responseEmbed("ArchiveTune Bot · "+title, description, ResponseColorError)
}

// WorkflowEmbed creates the richer status response used by moderation workflows.
// Bot and guild metadata are read from discordgo's ready/state cache when available.
func WorkflowEmbed(s *discordgo.Session, guildID, title, description string, color int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Author: &discordgo.MessageEmbedAuthor{
			Name: "ArchiveTune Bot",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "ArchiveTune",
		},
	}

	if s == nil || s.State == nil {
		return embed
	}

	s.State.RLock()
	botUser := s.State.User
	s.State.RUnlock()
	if botUser != nil {
		if botUser.Username != "" {
			embed.Author.Name = botUser.Username
		}
		embed.Author.IconURL = botUser.AvatarURL("64")
	}

	guild, err := s.State.Guild(guildID)
	if err == nil && guild != nil {
		if guild.Name != "" {
			embed.Footer.Text = guild.Name
		}
		embed.Footer.IconURL = guild.IconURL("64")
	}
	return embed
}

func responseEmbed(title, description string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "ArchiveTune Bot",
		},
	}
}
