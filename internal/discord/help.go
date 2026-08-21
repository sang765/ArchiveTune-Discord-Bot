package discord

import "github.com/bwmarrin/discordgo"

// HelpEmbed returns the complete command reference shown by .help and /help.
func HelpEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "ArchiveTune Bot Help",
		Description: "ArchiveTune Bot commands for managing issues and suggestion Forum Channel posts.",
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "General",
				Value:  "`.help` — show this command list",
				Inline: false,
			},
			{
				Name: "Issues prefix commands",
				Value: "`.solved` — apply `Solved`, lock, and rename to `[SOLVED] ...`\n" +
					"`.false` or `.false-report` — apply `False report`, lock, and rename to `[FALSE REPORT] ...`",
				Inline: false,
			},
			{
				Name: "Suggestion prefix commands",
				Value: "`.accept` — replace all tags with `Accept`, close, lock, and rename to `[ACCEPTED] ...`\n" +
					"`.accepted` — alias for `.accept`\n" +
					"`.dupe <post link>` — replace all tags with `Duplicate`, close, lock, and rename to `[DUPLICATE] ...`\n" +
					"`.done` — replace all tags with `Done`, close, lock, and rename to `[DONE] ...`\n" +
					"`.in-progress` — replace all tags with `In Progress...`, close, lock, and rename to `[IN PROGRESS] ...`\n" +
					"`.exist` — replace all tags with `Already exist`, close, lock, and rename to `[ALREADY EXIST] ...`\n" +
					"`.reject <reason>` — replace all tags with `Reject`, close, lock, rename, and send the reason\n" +
					"`.rejected` — alias for the generic `Reject` workflow\n" +
					"`.tba` or `.tbd` — replace all tags without closing, locking, or renaming",
				Inline: false,
			},
			{
				Name:   "Additional prefix commands",
				Value:  "`.maybe`, `.duplicate`, `.already-exist`, `.problem`, `.question`, `.stable`, `.nightly`, `.meta` — apply the configured tag, lock, and add the matching title prefix",
				Inline: false,
			},
			{
				Name: "Slash commands",
				Value: "`/help` — show this command list\n" +
					"`/forum-sync <channel>` — sync Forum Channel tags and settings\n" +
					"`/fix-suggestion` — apply `Maybe` to untagged suggestion posts\n" +
					"`/tag-add <tag> [post_id]` — add a configured tag\n" +
					"`/tag-remove <tag> [post_id]` — remove a configured tag\n" +
					"`/post-rename <name> [post_id]` — rename a post\n" +
					"`/post-state <open|close|lock|unlock> [post_id]` — change post state",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "ArchiveTune Bot · Prefix commands require Message Content Intent and moderator access.",
		},
	}
}
