package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/config"
	forumdiscord "github.com/sang765/discord-forum-bot/internal/discord"
	"github.com/sang765/discord-forum-bot/internal/media"
)

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "help",
		Description: "Show all bot commands",
	},
	{
		Name:        "ytd",
		Description: "Download YouTube or YouTube Music video, audio, or thumbnail",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "url", Description: "YouTube or YouTube Music URL", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "type", Description: "Media type", Type: discordgo.ApplicationCommandOptionString, Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "video", Value: "video"},
				{Name: "audio", Value: "audio"},
				{Name: "thumbnail", Value: "thumbnail"},
			}},
			{Name: "quality", Description: "Format ID or best; omit to list available formats", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
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
	mediaWorkDir := os.Getenv("MEDIA_WORK_DIR")
	if mediaWorkDir == "" {
		mediaWorkDir = ".tools/media-work"
	}
	if err := os.MkdirAll(mediaWorkDir, 0o755); err != nil {
		log.Fatalf("create media workspace: %v", err)
	}
	ytdDownloader := media.NewDownloader(os.Getenv("YTDLP_BIN"), os.Getenv("FFMPEG_BIN"), mediaWorkDir)
	ytdDownloader.BlockPlaylistAlbumDownload = cfg.YTD.BlockPlaylistAlbumDownloadEnabled()
	ytdSelections := media.NewSelectionStore(5 * time.Minute)

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
		if interaction.Type == discordgo.InteractionMessageComponent {
			handleYTDComponent(s, interaction, ytdSelections, ytdDownloader)
			return
		}
		if interaction.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if interaction.ApplicationCommandData().Name != "help" && interaction.ApplicationCommandData().Name != "ytd" && !forumdiscord.HasModeratorAccess(interaction, cfg) {
			respond(s, interaction, "You need `Manage Threads`, `Administrator`, or a configured moderator role to use this command.", true)
			return
		}
		handleCommand(s, interaction, manager, cfg, ytdDownloader, ytdSelections)
	})
	session.AddHandler(func(s *discordgo.Session, message *discordgo.MessageCreate) {
		logMessageEvent(message)
		if message == nil || message.Author == nil || message.Author.Bot || message.GuildID != cfg.GuildID {
			return
		}
		if strings.EqualFold(strings.TrimSpace(message.Content), ".help") {
			if _, err := s.ChannelMessageSendEmbed(message.ChannelID, forumdiscord.HelpEmbed()); err != nil {
				log.Println("send prefix help embed:", err)
			}
			return
		}
		if request, matched, valid, parseErr := media.ParseYTDCommandWithCollectionPolicy(message.Content, cfg.YTD.BlockPlaylistAlbumDownloadEnabled()); matched {
			if !valid {
				sendInfoMessage(s, message.ChannelID, "Usage", parseErr.Error())
				return
			}
			go handleYTDRequest(s, message.ChannelID, message.Author.ID, request, ytdDownloader, ytdSelections)
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

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate, manager *forumdiscord.Manager, cfg *config.Config, ytdDownloader *media.Downloader, ytdSelections *media.SelectionStore) {
	if err := deferInteraction(s, i); err != nil {
		log.Printf("defer slash command interaction: %v", err)
		return
	}
	data := i.ApplicationCommandData()
	switch data.Name {
	case "ytd":
		request := media.Request{URL: optionString(data.Options, "url"), Type: media.MediaType(optionString(data.Options, "type")), Quality: optionString(data.Options, "quality")}
		if err := media.ValidateRequestWithCollectionPolicy(request, cfg.YTD.BlockPlaylistAlbumDownloadEnabled()); err != nil {
			editInteraction(s, i, err.Error(), true)
			return
		}
		go handleYTDInteraction(s, i, request, ytdDownloader, ytdSelections)
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

func formatRejectReason(reason string) string {
	reason = strings.ReplaceAll(reason, "`", "'")
	return strings.ReplaceAll(reason, "@", "@\u200b")
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
		return fmt.Sprintf("The post was tagged `Reject`, closed and locked, and renamed to **%s**.\n\nReason for this suggestion has been rejected:\n```text\n%s\n```", postName, formatRejectReason(reason))
	}
	return details
}

func handleYTDRequest(s *discordgo.Session, channelID, requesterID string, request media.Request, downloader *media.Downloader, selections *media.SelectionStore) {
	info, err := downloader.Inspect(context.Background(), request)
	if err != nil {
		sendErrorMessage(s, channelID, "Media inspection failed", err.Error())
		return
	}
	if request.Quality == "" && request.Type != media.MediaThumbnail {
		sendYTDSelectionMessage(s, channelID, requesterID, request, info, selections)
		return
	}

	sendInfoMessage(s, channelID, "Preparing media", fmt.Sprintf("Preparing `%s` download for **%s**. Please wait...", request.Type, info.Title))
	result, err := downloader.DownloadAndUpload(context.Background(), request, info)
	if err != nil {
		sendErrorMessage(s, channelID, "Media download failed", err.Error())
		return
	}
	if _, err := s.ChannelMessageSendEmbed(channelID, mediaResultEmbed(result)); err != nil {
		log.Printf("send media result embed: %v", err)
	}
}

func handleYTDInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, request media.Request, downloader *media.Downloader, selections *media.SelectionStore) {
	info, err := downloader.Inspect(context.Background(), request)
	if err != nil {
		editInteraction(s, i, "Media inspection failed: "+err.Error(), true)
		return
	}
	if request.Quality == "" && request.Type != media.MediaThumbnail {
		sendYTDSelectionInteraction(s, i, requesterID(i), request, info, selections)
		return
	}
	result, err := downloader.DownloadAndUpload(context.Background(), request, info)
	if err != nil {
		editInteraction(s, i, "Media download failed: "+err.Error(), true)
		return
	}
	edited := mediaResultEmbed(result)
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{edited}}); err != nil {
		log.Printf("edit /ytd interaction response: %v", err)
	}
}

func sendMediaInfoMessage(s *discordgo.Session, channelID string, info media.Info, mediaType media.MediaType, summary string) {
	if _, err := s.ChannelMessageSendEmbed(channelID, mediaInfoEmbed(info, summary, mediaType)); err != nil {
		log.Printf("send media info embed: %v", err)
	}
}

func mediaInfoEmbed(info media.Info, summary string, mediaType media.MediaType) *discordgo.MessageEmbed {
	artist := safeText(info.Artist)
	description := fmt.Sprintf("**Type:** `%s`\n**Artist:** `%s`\n**Uploader:** `%s`\n**Duration:** `%s`\n\n%s", mediaType, artist, safeText(info.Uploader), formatDuration(info.Duration), summary)
	embed := forumdiscord.InfoEmbed("Media quality selection", description)
	if info.Title != "" {
		embed.Title = "Media quality selection · " + info.Title
	}
	if info.Thumbnail != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: info.Thumbnail}
	}
	return embed
}

func mediaResultEmbed(result media.Result) *discordgo.MessageEmbed {
	embed := forumdiscord.SuccessEmbed("Media ready", fmt.Sprintf("[Download `%s`](%s)\n\nThis temporary link expires after approximately 3 days.", result.FileName, result.DownloadURL))
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: "Title", Value: safeText(result.Info.Title), Inline: false},
		{Name: "Artist", Value: safeText(result.Info.Artist), Inline: true},
		{Name: "Uploader", Value: safeText(result.Info.Uploader), Inline: true},
		{Name: "Duration", Value: formatDuration(result.Info.Duration), Inline: true},
		{Name: "File", Value: fmt.Sprintf("`%s` · %s", result.FileName, formatBytes(result.FileSize)), Inline: false},
	}
	if result.Info.Thumbnail != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: result.Info.Thumbnail}
	}
	return embed
}

func editInteractionEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, isError bool) {
	if isError {
		embed.Color = forumdiscord.ResponseColorError
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{embed}}); err != nil {
		log.Printf("edit slash command embed response: %v", err)
	}
}

func safeText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	return value
}

func formatDuration(seconds float64) string {
	if seconds <= 0 {
		return "Unknown"
	}
	total := int64(seconds)
	hours := total / 3600
	minutes := (total % 3600) / 60
	remaining := total % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", hours, minutes, remaining)
	}
	return fmt.Sprintf("%dm %02ds", minutes, remaining)
}

func formatBytes(size int64) string {
	if size <= 0 {
		return "Unknown size"
	}
	const unit = 1024.0
	if float64(size) >= unit*unit*unit {
		return fmt.Sprintf("%.2f GB", float64(size)/(unit*unit*unit))
	}
	if float64(size) >= unit*unit {
		return fmt.Sprintf("%.2f MB", float64(size)/(unit*unit))
	}
	return fmt.Sprintf("%.2f KB", float64(size)/unit)
}

func sendYTDSelectionMessage(s *discordgo.Session, channelID, requesterID string, request media.Request, info media.Info, selections *media.SelectionStore) {
	selectionID, err := selections.Create(media.Selection{Request: request, Info: info, ChannelID: channelID, UserID: requesterID})
	if err != nil {
		sendErrorMessage(s, channelID, "Media selection failed", err.Error())
		return
	}
	selection := media.Selection{Request: request, Info: info, ChannelID: channelID, UserID: requesterID}
	selectionEmbed := mediaInfoEmbed(info, "Choose a quality from the menu below, then press **Download**. The download will not start until you confirm.", request.Type)
	message, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{selectionEmbed},
		Components: ytdComponents(selectionID, selection),
	})
	if err != nil {
		selections.Delete(selectionID, requesterID)
		log.Printf("send ytd selection message: %v", err)
		return
	}
	if message != nil {
		selection.MessageID = message.ID
		// Store the message ID for traceability; the selection ID remains in component custom IDs.
		selections.SetMessageID(selectionID, requesterID, message.ID)
	}
}

func sendYTDSelectionInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, requesterID string, request media.Request, info media.Info, selections *media.SelectionStore) {
	selectionID, err := selections.Create(media.Selection{Request: request, Info: info, ChannelID: i.ChannelID, UserID: requesterID})
	if err != nil {
		editInteraction(s, i, "Media selection failed: "+err.Error(), true)
		return
	}
	selection := media.Selection{Request: request, Info: info, ChannelID: i.ChannelID, UserID: requesterID}
	selectionEmbed := mediaInfoEmbed(info, "Choose a quality from the menu below, then press **Download**. The download will not start until you confirm.", request.Type)
	message, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{selectionEmbed},
		Components: componentPointer(ytdComponents(selectionID, selection)),
	})
	if err != nil {
		selections.Delete(selectionID, requesterID)
		log.Printf("edit /ytd quality selection response: %v", err)
		return
	}
	if message != nil {
		selections.SetMessageID(selectionID, requesterID, message.ID)
	}
}

func handleYTDComponent(s *discordgo.Session, i *discordgo.InteractionCreate, selections *media.SelectionStore, downloader *media.Downloader) {
	data := i.MessageComponentData()
	parts := strings.SplitN(data.CustomID, "_", 3)
	if len(parts) != 3 || parts[0] != "ytd" {
		return
	}
	userID := requesterID(i)
	selectionID := parts[2]
	switch parts[1] {
	case "select":
		if len(data.Values) != 1 {
			componentError(s, i, "Please choose exactly one quality.")
			return
		}
		selection, ok := selections.SetQuality(selectionID, userID, data.Values[0])
		if !ok {
			componentError(s, i, "This media selection expired or belongs to another user. Run `.ytd` again.")
			return
		}
		updated := mediaInfoEmbed(selection.Info, fmt.Sprintf("Selected quality: `%s`. Press **Download** to start, or **Cancel** to discard this request.", displayQuality(selection.Quality)), selection.Request.Type)
		respondComponentUpdate(s, i, updated, ytdComponents(selectionID, selection))
	case "download":
		selection, ok := selections.Take(selectionID, userID)
		if !ok {
			componentError(s, i, "Choose a quality first, or run `.ytd` again because this selection expired.")
			return
		}
		if err := respondComponentDeferredUpdate(s, i); err != nil {
			log.Printf("defer ytd download component: %v", err)
			return
		}
		request := selection.Request
		request.Quality = selection.Quality
		if err := editYTDMessage(s, selection.ChannelID, selection.MessageID, mediaInfoEmbed(selection.Info, fmt.Sprintf("Downloading `%s` quality `%s`... Please wait.", request.Type, displayQuality(request.Quality)), request.Type), nil); err != nil {
			log.Printf("mark ytd message downloading: %v", err)
		}
		go finishYTDComponentDownload(s, selection, request, downloader)
	case "cancel":
		selection, ok := selections.Get(selectionID)
		if !ok || (selection.UserID != "" && selection.UserID != userID) {
			componentError(s, i, "This media selection expired or belongs to another user.")
			return
		}
		selections.Delete(selectionID, userID)
		respondComponentUpdate(s, i, forumdiscord.InfoEmbed("Media download cancelled", "The download request was cancelled."), nil)
	}
}

func finishYTDComponentDownload(s *discordgo.Session, selection media.Selection, request media.Request, downloader *media.Downloader) {
	result, err := downloader.DownloadAndUpload(context.Background(), request, selection.Info)
	if err != nil {
		if editErr := editYTDMessage(s, selection.ChannelID, selection.MessageID, forumdiscord.ErrorEmbed("Media download failed", err.Error()), nil); editErr != nil {
			log.Printf("edit failed ytd message: %v", editErr)
		}
		return
	}
	if err := editYTDMessage(s, selection.ChannelID, selection.MessageID, mediaResultEmbed(result), nil); err != nil {
		log.Printf("edit completed ytd message: %v", err)
	}
}

func ytdComponents(selectionID string, selection media.Selection) []discordgo.MessageComponent {
	options := make([]discordgo.SelectMenuOption, 0, 16)
	options = append(options, discordgo.SelectMenuOption{Label: "Best available quality", Value: "best", Description: "Let yt-dlp choose the highest quality"})
	seen := map[string]struct{}{"best": {}}
	for _, format := range selection.Info.Formats {
		if format.ID == "" || format.ID == "storyboard" {
			continue
		}
		isVideo := format.VCodec != "" && format.VCodec != "none"
		isAudio := format.ACodec != "" && format.ACodec != "none"
		if selection.Request.Type == media.MediaVideo && !isVideo {
			continue
		}
		if selection.Request.Type == media.MediaAudio && !isAudio {
			continue
		}
		if _, ok := seen[format.ID]; ok {
			continue
		}
		seen[format.ID] = struct{}{}
		options = append(options, discordgo.SelectMenuOption{Label: truncateComponentText(media.FormatLabel(format), 100), Value: format.ID, Description: truncateComponentText(formatDescription(format), 100)})
		if len(options) == 25 {
			break
		}
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{MenuType: discordgo.StringSelectMenu, CustomID: "ytd_select_" + selectionID, Placeholder: "Choose a quality", MinValues: intPointer(1), MaxValues: 1, Options: options, Disabled: false},
		}},
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Download", Style: discordgo.SuccessButton, CustomID: "ytd_download_" + selectionID, Disabled: selection.Quality == ""},
			discordgo.Button{Label: "Cancel", Style: discordgo.DangerButton, CustomID: "ytd_cancel_" + selectionID},
		}},
	}
}

func formatDescription(format media.Format) string {
	if format.Width > 0 && format.Height > 0 {
		return fmt.Sprintf("%dx%d · %s", format.Width, format.Height, format.Ext)
	}
	if format.Bitrate > 0 {
		return fmt.Sprintf("%.0f kbps · %s", format.Bitrate, format.Ext)
	}
	return format.Ext
}

func displayQuality(quality string) string {
	if strings.EqualFold(quality, "best") {
		return "best available"
	}
	return quality
}

func truncateComponentText(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}

func intPointer(value int) *int {
	return &value
}

func componentPointer(components []discordgo.MessageComponent) *[]discordgo.MessageComponent {
	return &components
}

func requesterID(i *discordgo.InteractionCreate) string {
	if i == nil {
		return ""
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func respondComponentUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) {
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseUpdateMessage, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}, Components: components}}); err != nil {
		log.Printf("update ytd component message: %v", err)
	}
}

func respondComponentDeferredUpdate(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})
}

func componentError(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{forumdiscord.ErrorEmbed("Media selection failed", content)}, Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
		log.Printf("respond ytd component error: %v", err)
	}
}

func editYTDMessage(s *discordgo.Session, channelID, messageID string, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	if messageID == "" {
		return nil
	}
	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	edit := discordgo.NewMessageEdit(channelID, messageID).SetEmbed(embed)
	edit.Components = &components
	_, err := s.ChannelMessageEditComplex(edit)
	return err
}
