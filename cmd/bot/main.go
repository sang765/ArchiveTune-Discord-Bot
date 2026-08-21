package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/config"
	forumdiscord "github.com/sang765/discord-forum-bot/internal/discord"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "help",
		Description: "Show all bot commands",
	},
	{
		Name:        "forum-sync",
		Description: "Sync guidelines, tags, and required-tag settings for a Forum Channel",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "channel", Description: "Forum Channel to sync", Type: discordgo.ApplicationCommandOptionChannel, Required: true},
		},
	},
	{
		Name:        "fix-suggestion",
		Description: "Apply the Maybe tag to suggestion posts without tags",
	},
	{
		Name:        "tag-add",
		Description: "Add a tag to the current or specified post",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "tag", Description: "Configured tag name", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "post_id", Description: "Post ID; leave empty to use the current post", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
	{
		Name:        "tag-remove",
		Description: "Remove a tag from the current or specified post",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "tag", Description: "Configured tag name", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "post_id", Description: "Post ID; leave empty to use the current post", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
	{
		Name:        "post-rename",
		Description: "Rename a managed Forum Channel post",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "name", Description: "New post title", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "post_id", Description: "Post ID; leave empty to use the current post", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
	{
		Name:        "post-state",
		Description: "Open or close, and lock or unlock, a post",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "state", Description: "State to apply", Type: discordgo.ApplicationCommandOptionString, Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "open", Value: "open"},
				{Name: "close", Value: "close"},
				{Name: "lock", Value: "lock"},
				{Name: "unlock", Value: "unlock"},
			}},
			{Name: "post_id", Description: "Post ID; leave empty to use the current post", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
}

func main() {
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		log.Fatalf("create Discord session: %v", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentMessageContent
	manager := forumdiscord.NewManager(session, cfg)

	session.AddHandler(func(s *discordgo.Session, ready *discordgo.Ready) {
		log.Printf("logged in as %s#%s", ready.User.Username, ready.User.Discriminator)
		if _, err := s.ApplicationCommandBulkOverwrite(ready.User.ID, cfg.GuildID, commands); err != nil {
			log.Printf("register slash commands: %v", err)
			return
		}
		if err := manager.SyncConfiguredChannels(); err != nil {
			log.Printf("initial forum sync: %v", err)
			return
		}
		log.Printf("synced %d managed Forum Channel(s)", len(cfg.Channels))
	})
	session.AddHandler(func(s *discordgo.Session, thread *discordgo.ThreadCreate) {
		if thread == nil || thread.GuildID != cfg.GuildID || thread.ParentID != forumdiscord.SuggestionChannelID {
			return
		}
		changed, err := manager.MaybeTagIfMissing(thread.ID)
		if err != nil {
			log.Printf("auto-tag suggestion post %s: %v", thread.ID, err)
			return
		}
		if changed {
			log.Printf("auto-tagged new suggestion post %s with Maybe", thread.ID)
		}
	})
	session.AddHandler(func(s *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if interaction.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if interaction.ApplicationCommandData().Name != "help" && !forumdiscord.HasModeratorAccess(interaction, cfg) {
			respond(s, interaction, "You need `Manage Threads`, `Administrator`, or a configured moderator role to use this command.", true)
			return
		}
		handleCommand(s, interaction, manager, cfg)
	})
	session.AddHandler(func(s *discordgo.Session, message *discordgo.MessageCreate) {
		logMessageEvent(message)
		if message == nil || message.Author == nil || message.Author.Bot || message.GuildID != cfg.GuildID {
			return
		}
		if strings.EqualFold(strings.TrimSpace(message.Content), ".help") {
			if _, err := s.ChannelMessageSendEmbed(message.ChannelID, forumdiscord.HelpEmbed()); err != nil {
				log.Printf("send prefix help embed: %v", err)
			}
			return
		}
		prefixCandidate := strings.HasPrefix(strings.TrimSpace(message.Content), ".")
		if reason, matched, valid := forumdiscord.ParseRejectCommand(message.Content); matched {
			if !valid {
				_, _ = s.ChannelMessageSend(message.ChannelID, "Usage: `.reject <reason>` — the reason is required and must be at most 1,000 characters.")
				return
			}
			if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
				_, _ = s.ChannelMessageSend(message.ChannelID, "You need `Manage Threads`, `Administrator`, or a configured moderator role to use `.reject`.")
				return
			}
			statusAction := forumdiscord.SuggestionStatusAction{Command: ".reject", TagName: "Reject", TitlePrefix: "[REJECTED]"}
			authorMention, authorErr := manager.ThreadAuthorMention(message.ChannelID)
			if authorErr != nil {
				log.Printf("get author for %s: %v", message.ChannelID, authorErr)
			}
			updated, err := manager.ApplySuggestionStatusAction(message.ChannelID, statusAction)
			if err != nil {
				_, _ = s.ChannelMessageSend(message.ChannelID, "Could not process `.reject`: "+err.Error())
				return
			}
			messageText := fmt.Sprintf("Rejected the suggestion: replaced its tags with `Reject`, closed and locked the post, and renamed it to **%s**. Reason: %s", updated.Name, reason)
			if authorMention != "" {
				messageText = authorMention + " " + messageText
			}
			_, _ = s.ChannelMessageSend(message.ChannelID, messageText)
			return
		}

		if statusAction, link, matched, valid := forumdiscord.ParseSuggestionStatusCommand(message.Content); matched {
			if !valid {
				if statusAction.Command == ".dupe" {
					_, _ = s.ChannelMessageSend(message.ChannelID, "Usage: `.dupe <Discord post link or message link>`")
				} else {
					_, _ = s.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Usage: `%s` — send this command directly in the post you want to process.", statusAction.Command))
				}
				return
			}
			if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
				_, _ = s.ChannelMessageSend(message.ChannelID, "You need `Manage Threads`, `Administrator`, or a configured moderator role to use suggestion status commands.")
				return
			}
			targetID := message.ChannelID
			if link != "" {
				var err error
				targetID, err = forumdiscord.ParseDiscordPostLink(link, cfg.GuildID)
				if err != nil {
					_, _ = s.ChannelMessageSend(message.ChannelID, "Could not parse post link: "+err.Error())
					return
				}
			}
			authorMention, authorErr := manager.ThreadAuthorMention(targetID)
			if authorErr != nil {
				log.Printf("get author for %s: %v", targetID, authorErr)
			}
			updated, err := manager.ApplySuggestionStatusAction(targetID, statusAction)
			if err != nil {
				_, _ = s.ChannelMessageSend(message.ChannelID, "Could not process "+statusAction.Command+": "+err.Error())
				return
			}
			messageText := fmt.Sprintf("Applied `%s`: replaced all tags with `%s`, closed and locked the post, and renamed it to **%s**.", statusAction.Command, statusAction.TagName, updated.Name)
			if authorMention != "" {
				messageText = authorMention + " " + messageText
			}
			_, _ = s.ChannelMessageSend(message.ChannelID, messageText)
			return
		}

		action, ok := forumdiscord.ParsePrefixCommand(message.Content)
		originalCommand := ""
		if !ok && cfg.PrefixAutocorrect {
			var correctedCommand string
			action, correctedCommand, ok = forumdiscord.GuessPrefixCommand(message.Content, cfg.PrefixMaxDistance)
			if ok {
				originalCommand = strings.TrimSpace(message.Content)
				log.Printf("autocorrected prefix %q to %q", originalCommand, correctedCommand)
			}
		}
		if !ok {
			if prefixCandidate {
				log.Printf("prefix command not recognized: guild_id=%s channel_id=%s content=%q", message.GuildID, message.ChannelID, strings.TrimSpace(message.Content))
			}
			return
		}
		log.Printf("prefix command recognized: guild_id=%s channel_id=%s command=%s", message.GuildID, message.ChannelID, action.Command)
		if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
			log.Printf("prefix command denied: channel_id=%s command=%s member_permissions=%d roles=%v", message.ChannelID, action.Command, memberPermissions(message), memberRoleIDs(message))
			_, _ = s.ChannelMessageSend(message.ChannelID, "You need `Manage Threads`, `Administrator`, or a configured moderator role to use prefix commands.")
			return
		}
		authorMention := ""
		if action.Command == ".solved" || action.Command == ".false" || action.Command == ".false-report" {
			var authorErr error
			authorMention, authorErr = manager.ThreadAuthorMention(message.ChannelID)
			if authorErr != nil {
				log.Printf("get author for %s: %v", message.ChannelID, authorErr)
			}
		}
		log.Printf("prefix command applying: channel_id=%s command=%s", message.ChannelID, action.Command)
		updated, err := manager.ApplyPrefixAction(message.ChannelID, action)
		if err != nil {
			log.Printf("prefix command failed: channel_id=%s command=%s error=%v", message.ChannelID, action.Command, err)
			_, _ = s.ChannelMessageSend(message.ChannelID, "Could not process "+action.Command+": "+err.Error())
			return
		}
		messageText := fmt.Sprintf("Applied `%s`: set tag `%s`, locked the post, and renamed it to **%s**.", action.Command, action.TagName, updated.Name)
		if action.Command == ".tba" || action.Command == ".tbd" {
			messageText = fmt.Sprintf("Applied `%s`: removed all previous tags and kept only `%s`; the post was not locked, closed, or renamed.", action.Command, action.TagName)
		}
		if authorMention != "" {
			messageText = authorMention + " " + messageText
		}
		if originalCommand != "" {
			messageText = fmt.Sprintf("I corrected `%s` to `%s`. %s", originalCommand, action.Command, messageText)
		}
		_, _ = s.ChannelMessageSend(message.ChannelID, messageText)
		log.Printf("prefix command completed: channel_id=%s command=%s post_name=%q", message.ChannelID, action.Command, updated.Name)
	})

	if err := session.Open(); err != nil {
		log.Fatalf("open Discord gateway: %v", err)
	}
	defer session.Close()
	log.Println("bot is running; press Ctrl+C to stop")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
}

func logMessageEvent(message *discordgo.MessageCreate) {
	if message == nil {
		log.Printf("message event received: message=nil")
		return
	}
	authorID := ""
	authorBot := false
	if message.Author != nil {
		authorID = message.Author.ID
		authorBot = message.Author.Bot
	}
	content := strings.TrimSpace(message.Content)
	prefixCandidate := strings.HasPrefix(content, ".")
	if !prefixCandidate && os.Getenv("DEBUG_MESSAGE_EVENTS") != "1" {
		return
	}
	log.Printf("message event received: message_id=%s guild_id=%s channel_id=%s author_id=%s author_bot=%t content_len=%d starts_with_dot=%t", message.ID, message.GuildID, message.ChannelID, authorID, authorBot, len([]rune(content)), prefixCandidate)
	if prefixCandidate {
		log.Printf("message content for prefix candidate: %q", content)
	}
}

func memberPermissions(message *discordgo.MessageCreate) uint64 {
	if message == nil || message.Member == nil {
		return 0
	}
	return uint64(message.Member.Permissions)
}

func memberRoleIDs(message *discordgo.MessageCreate) []string {
	if message == nil || message.Member == nil {
		return nil
	}
	return message.Member.Roles
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate, manager *forumdiscord.Manager, cfg *config.Config) {
	if err := deferInteraction(s, i); err != nil {
		log.Printf("defer slash command interaction: %v", err)
		return
	}
	data := i.ApplicationCommandData()
	switch data.Name {
	case "help":
		embed := forumdiscord.HelpEmbed()
		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}}); err != nil {
			log.Printf("edit /help interaction response: %v", err)
		}
	case "forum-sync":
		channelID := optionString(data.Options, "channel")
		channel, err := manager.SyncChannel(channelID)
		if err != nil {
			editInteraction(s, i, err.Error(), true)
			return
		}
		editInteraction(s, i, fmt.Sprintf("Synced Forum Channel %s with %d tags.", channel.Mention(), len(channel.AvailableTags)), false)
	case "fix-suggestion":
		scanned, fixed, err := manager.FixSuggestion()
		content := fmt.Sprintf("Scanned %d suggestion posts and applied the `Maybe` tag to %d posts that had no tags.", scanned, fixed)
		if err != nil {
			content = "Could not fix suggestion posts: " + err.Error()
		}
		if _, editErr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); editErr != nil {
			log.Printf("edit /fix-suggestion interaction response: %v", editErr)
		}
	case "tag-add":
		postID := postIDFrom(i, data.Options)
		updated, err := manager.ApplyTag(postID, optionString(data.Options, "tag"))
		if err != nil {
			editInteraction(s, i, err.Error(), true)
			return
		}
		editInteraction(s, i, fmt.Sprintf("Added the tag to post %s.", updated.Mention()), false)
	case "tag-remove":
		postID := postIDFrom(i, data.Options)
		updated, err := manager.RemoveTag(postID, optionString(data.Options, "tag"))
		if err != nil {
			editInteraction(s, i, err.Error(), true)
			return
		}
		editInteraction(s, i, fmt.Sprintf("Removed the tag from post %s.", updated.Mention()), false)
	case "post-rename":
		postID := postIDFrom(i, data.Options)
		updated, err := manager.RenameThread(postID, optionString(data.Options, "name"))
		if err != nil {
			editInteraction(s, i, err.Error(), true)
			return
		}
		editInteraction(s, i, fmt.Sprintf("Renamed the post to **%s**.", updated.Name), false)
	case "post-state":
		postID := postIDFrom(i, data.Options)
		state := optionString(data.Options, "state")
		if state != "open" && state != "close" && state != "lock" && state != "unlock" {
			editInteraction(s, i, "Invalid post state.", true)
			return
		}
		updated, err := manager.SetThreadState(postID, state)
		if err != nil {
			editInteraction(s, i, err.Error(), true)
			return
		}
		editInteraction(s, i, fmt.Sprintf("Set post state to `%s` for post %s.", state, updated.Mention()), false)
	default:
		editInteraction(s, i, "Unknown slash command.", true)
	}

}

func optionString(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if option.Name == name {
			return strings.TrimSpace(option.StringValue())
		}
	}
	return ""
}

func postIDFrom(i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	if postID := optionString(options, "post_id"); postID != "" {
		return postID
	}
	return i.ChannelID
}

func deferInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func editInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string, _ bool) {
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content}); err != nil {
		log.Printf("edit slash command response: %v", err)
	}
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content, Flags: flags},
	}); err != nil {
		log.Printf("interaction response: %v", err)
	}
}
