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
