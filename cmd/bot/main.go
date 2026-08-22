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
		log.Printf("ArchiveTune Bot logged in as %s#%s", ready.User.Username, ready.User.Discriminator)
		if err := forumdiscord.SetArchiveTunePresence(s); err != nil {
			log.Printf("set ArchiveTune Bot presence: %v", err)
		}
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
				sendInfoMessage(s, message.ChannelID, "Usage", "`.reject <reason>` — the reason is required and must be at most 1,000 characters.")
				return
			}
			if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
				sendErrorMessage(s, message.ChannelID, "Permission denied", "You need `Manage Threads`, `Administrator`, or a configured moderator role to use `.reject`.")
				return
			}
			statusAction := forumdiscord.SuggestionStatusAction{Command: ".reject", TagName: "Reject", TitlePrefix: "[REJECTED]"}
			authorMention, authorErr := manager.ThreadAuthorMention(message.ChannelID)
			if authorErr != nil {
				log.Printf("get author for %s: %v", message.ChannelID, authorErr)
			}
			updated, err := manager.ApplySuggestionStatusAction(message.ChannelID, statusAction)
			if err != nil {
				sendErrorMessage(s, message.ChannelID, "Command failed", "Could not process `.reject`: "+err.Error())
				return
			}
			messageText := fmt.Sprintf("The post was tagged `Reject`, closed and locked, and renamed to **%s**. Reason: %s", updated.Name, reason)
			sendWorkflowMessage(s, message.ChannelID, message.GuildID, authorMention, statusAction.Command, updated.Name, messageText, reason, nil)
			return
		}

		if statusAction, link, matched, valid := forumdiscord.ParseSuggestionStatusCommand(message.Content); matched {
			if !valid {
				if statusAction.Command == ".dupe" {
					sendInfoMessage(s, message.ChannelID, "Usage", "`.dupe <Discord post link or message link>`")
				} else {
					sendInfoMessage(s, message.ChannelID, "Usage", fmt.Sprintf("`%s` — send this command directly in the post you want to process.", statusAction.Command))
				}
				return
			}
			if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
				sendErrorMessage(s, message.ChannelID, "Permission denied", "You need `Manage Threads`, `Administrator`, or a configured moderator role to use suggestion status commands.")
				return
			}
			targetID := message.ChannelID
			var duplicateReference *duplicateReferenceData
			if link != "" {
				referencedID, err := forumdiscord.ParseDiscordPostLink(link, cfg.GuildID)
				if err != nil {
					sendErrorMessage(s, message.ChannelID, "Invalid post link", "Could not parse post link: "+err.Error())
					return
				}
				referencedPost, referencedCfg, err := manager.ManagedThread(referencedID)
				if err != nil {
					sendErrorMessage(s, message.ChannelID, "Invalid duplicate post", "Could not load the referenced suggestion post: "+err.Error())
					return
				}
				if !strings.EqualFold(strings.TrimSpace(referencedCfg.Name), "suggestion") {
					sendErrorMessage(s, message.ChannelID, "Invalid duplicate post", "The referenced post is not in the managed suggestion Forum Channel.")
					return
				}
				referencedAuthor, authorErr := manager.ThreadAuthorMention(referencedID)
				if authorErr != nil {
					log.Printf("get author for referenced post %s: %v", referencedID, authorErr)
				}
				duplicateReference = &duplicateReferenceData{
					name:          referencedPost.Name,
					link:          fmt.Sprintf("https://discord.com/channels/%s/%s", cfg.GuildID, referencedID),
					authorMention: referencedAuthor,
				}
			}
			authorMention, authorErr := manager.ThreadAuthorMention(targetID)
			if authorErr != nil {
				log.Printf("get author for %s: %v", targetID, authorErr)
			}
			updated, err := manager.ApplySuggestionStatusAction(targetID, statusAction)
			if err != nil {
				sendErrorMessage(s, message.ChannelID, "Command failed", "Could not process "+statusAction.Command+": "+err.Error())
				return
			}
			messageText := fmt.Sprintf("The post was tagged `%s`, closed and locked, and renamed to **%s**.", statusAction.TagName, updated.Name)
			sendWorkflowMessage(s, message.ChannelID, message.GuildID, authorMention, statusAction.Command, updated.Name, messageText, "", duplicateReference)
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
			sendErrorMessage(s, message.ChannelID, "Permission denied", "You need `Manage Threads`, `Administrator`, or a configured moderator role to use prefix commands.")
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
			sendErrorMessage(s, message.ChannelID, "Command failed", "Could not process "+action.Command+": "+err.Error())
			return
		}
		messageText := fmt.Sprintf("Applied `%s`: set tag `%s`, locked the post, and renamed it to **%s**.", action.Command, action.TagName, updated.Name)
		if action.Command == ".tba" || action.Command == ".tbd" {
			messageText = fmt.Sprintf("Applied `%s`: removed all previous tags and kept only `%s`; the post was not locked, closed, or renamed.", action.Command, action.TagName)
		}
		if originalCommand != "" {
			messageText = fmt.Sprintf("I corrected `%s` to `%s`. %s", originalCommand, action.Command, messageText)
		}
		if action.Command == ".solved" || action.Command == ".false" || action.Command == ".false-report" {
			sendWorkflowMessage(s, message.ChannelID, message.GuildID, authorMention, action.Command, updated.Name, messageText, "", nil)
		} else {
			sendSuccessMessage(s, message.ChannelID, "Command completed", messageText)
		}
		log.Printf("prefix command completed: channel_id=%s command=%s post_name=%q", message.ChannelID, action.Command, updated.Name)
	})

	if err := session.Open(); err != nil {
		log.Fatalf("open Discord gateway: %v", err)
	}
	defer session.Close()
	log.Println("ArchiveTune Bot is running; press Ctrl+C to stop")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("ArchiveTune Bot shutting down")
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
			editInteraction(s, i, "Could not fix suggestion posts: "+err.Error(), true)
			return
		}
		editInteraction(s, i, content, false)
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

func editInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string, isError bool) {
	embed := forumdiscord.SuccessEmbed("Command completed", content)
	if isError {
		embed = forumdiscord.ErrorEmbed("Command failed", content)
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}}); err != nil {
		log.Printf("edit slash command response: %v", err)
	}
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	var flags discordgo.MessageFlags
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	embed := forumdiscord.ErrorEmbed("Permission denied", content)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Flags: flags},
	}); err != nil {
		log.Printf("interaction response: %v", err)
	}
}

func sendInfoMessage(s *discordgo.Session, channelID, title, description string) {
	if _, err := s.ChannelMessageSendEmbed(channelID, forumdiscord.InfoEmbed(title, description)); err != nil {
		log.Printf("send info embed: %v", err)
	}
}

func sendSuccessMessage(s *discordgo.Session, channelID, title, description string) {
	if _, err := s.ChannelMessageSendEmbed(channelID, forumdiscord.SuccessEmbed(title, description)); err != nil {
		log.Printf("send success embed: %v", err)
	}
}

func sendErrorMessage(s *discordgo.Session, channelID, title, description string) {
	if _, err := s.ChannelMessageSendEmbed(channelID, forumdiscord.ErrorEmbed(title, description)); err != nil {
		log.Printf("send error embed: %v", err)
	}
}

type duplicateReferenceData struct {
	name          string
	link          string
	authorMention string
}

func sendWorkflowMessage(s *discordgo.Session, channelID, guildID, content, command, postName, details, reason string, duplicateReference *duplicateReferenceData) {
	title := "✅ Your suggestion has been updated."
	color := forumdiscord.ResponseColorInfo
	switch command {
	case ".solved":
		title = "✅ Your issue has been marked as `solved`!!!"
		color = forumdiscord.ResponseColorSuccess
	case ".false", ".false-report":
		title = "⚠️ Your issue has been marked as `false report`."
		color = forumdiscord.ResponseColorWarning
	case ".accept", ".accepted":
		title = "✅ Your suggestion has been marked as `accepted`."
		color = forumdiscord.ResponseColorSuccess
	case ".dupe":
		title = "♻️ Your suggestion has been marked as `duplicate`."
	case ".done":
		title = "✅ Your suggestion has been marked as `done`."
		color = forumdiscord.ResponseColorSuccess
	case ".in-progress":
		title = "🔄 Your suggestion is now `in progress`."
	case ".exist":
		title = "ℹ️ Your suggestion has been marked as `already exist`."
	case ".reject", ".rejected":
		title = "❌ Your suggestion has been `rejected`."
		color = forumdiscord.ResponseColorError
	}

	description := workflowDescription(command, postName, details, reason, duplicateReference)

	embed := forumdiscord.WorkflowEmbed(s, guildID, title, description, color)
	if _, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	}); err != nil {
		log.Printf("send workflow embed: %v", err)
	}
}

func workflowDescription(command, postName, details, reason string, duplicateReference *duplicateReferenceData) string {
	switch command {
	case ".false", ".false-report":
		return fmt.Sprintf("Applied `.false`: set tag `False report`, locked the post, and renamed it to **%s**.\n\nNote: You may be warned, muted, kicked, or even worse, banned if you create nonsensical or inaccurate issue posts, so be careful.", postName)
	case ".dupe":
		if duplicateReference != nil {
			return fmt.Sprintf("The post was tagged `Duplicate`, closed and locked, and renamed to **%s**.\n\n**What suggestion post has been duplicated?**\n**[\"%s\"](%s)** by %s", postName, duplicateReference.name, duplicateReference.link, duplicateReference.authorMention)
		}
	case ".reject", ".rejected":
		return fmt.Sprintf("The post was tagged `Reject`, closed and locked, and renamed to **%s**. \n\nReason for this suggestion has been rejected: \n**`%s`**", postName, reason)
	}
	return details
}
