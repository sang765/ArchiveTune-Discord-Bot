package discord

import "github.com/bwmarrin/discordgo"

const ArchiveTuneCustomStatus = "Why do I exist?"

func ArchiveTunePresence() discordgo.UpdateStatusData {
	return discordgo.UpdateStatusData{
		Status: string(discordgo.StatusDoNotDisturb),
		Activities: []*discordgo.Activity{
			{
				Name:  "Custom Status",
				Type:  discordgo.ActivityTypeCustom,
				State: ArchiveTuneCustomStatus,
			},
		},
	}
}

func SetArchiveTunePresence(session *discordgo.Session) error {
	return session.UpdateStatusComplex(ArchiveTunePresence())
}
