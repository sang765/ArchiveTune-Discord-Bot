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

var commands = append([]*discordgo.ApplicationCommand{
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
	{
		Name:        "close",
		Description: "Close and delete your suggestion post (OP only)",
	},
}, statusSlashCommands()...)

func main() {
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.yaml"
	}
	if _, statErr := os.Stat(configPath); statErr == nil {
		changed, defaultsErr := config.EnsureDefaults(configPath)
		if defaultsErr != nil {
			log.Fatalf("configuration defaults error: %v", defaultsErr)
		}
		if changed {
			log.Printf("added missing configuration defaults to %s", configPath)
		}
	} else if !os.IsNotExist(statErr) {
		log.Fatalf("configuration file error: %v", statErr)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	session, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		log.Fatalf("create Discord session: %v", err)
	}
	// No Message Content Intent: every command is an app (slash) command or an
	// interaction component, so the bot never needs to read message bodies.
	// GuildMessages is still needed for the MessageDelete cleanup handler.
	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions
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
		if _, err := manager.MaybeTagIfMissing(thread.ID); err != nil {
			log.Printf("auto-tag suggestion post %s: %v", thread.ID, err)
		}
		// Add a small delay to let Discord fully create the thread
		time.Sleep(500 * time.Millisecond)
		messages, err := s.ChannelMessages(thread.ID, 10, "", "", "")
		if err != nil {
			log.Printf("check messages for welcome in %s: %v", thread.ID, err)
			// Try sending welcome anyway - ChannelMessages might fail but sending works
			sendSuggestionWelcomeMessage(s, thread)
			return
		}
		botID := s.State.User.ID
		alreadyWelcomed := false
		for _, msg := range messages {
			if msg.Author != nil && msg.Author.ID == botID {
				alreadyWelcomed = true
				break
			}
		}
		if !alreadyWelcomed {
			sendSuggestionWelcomeMessage(s, thread)
		}
	})
	session.AddHandler(func(s *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if interaction.Type == discordgo.InteractionMessageComponent {
			data := interaction.MessageComponentData()
			if data.CustomID == "close_suggestion" {
				handleCloseSuggestionButton(s, interaction, cfg)
				return
			}
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
	session.AddHandler(func(s *discordgo.Session, event *discordgo.MessageDelete) {
		if event.Message == nil || event.Message.Author == nil || event.Message.Author.Bot {
			return
		}
		channel, err := s.Channel(event.Message.ChannelID)
		if err != nil {
			log.Printf("get channel for deleted message: %v", err)
			return
		}
		if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread {
			return
		}
		if channel.ParentID != forumdiscord.SuggestionChannelID {
			return
		}
		if event.Message.Author.ID != channel.OwnerID {
			return
		}
		if _, err := s.ChannelDelete(channel.ID); err != nil {
			log.Printf("delete thread %s: %v", channel.ID, err)
		} else {
			log.Printf("deleted thread %s because author %s deleted main post", channel.ID, event.Message.Author.ID)
		}
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

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate, manager *forumdiscord.Manager, cfg *config.Config, ytdDownloader *media.Downloader, ytdSelections *media.SelectionStore) {
	if err := deferInteraction(s, i); err != nil {
		log.Printf("defer slash command interaction: %v", err)
		return
	}
	data := i.ApplicationCommandData()
	if spec, ok := statusActionByName(data.Name); ok {
		handleStatusCommand(s, i, manager, cfg, spec)
		return
	}
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
	case "close":
		channelID := i.ChannelID
		channel, err := s.Channel(channelID)
		if err != nil {
			editInteraction(s, i, "Could not fetch post info: "+err.Error(), true)
			return
		}
		if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread {
			editInteraction(s, i, "This command can only be used inside a suggestion post.", true)
			return
		}
		if channel.ParentID != forumdiscord.SuggestionChannelID {
			editInteraction(s, i, "This command can only be used in the suggestion channel.", true)
			return
		}
		userID := requesterID(i)
		if channel.OwnerID != userID {
			editInteraction(s, i, "Only the post creator can use this command.", true)
			return
		}
		if _, err := s.ChannelDelete(channelID); err != nil {
			editInteraction(s, i, "Could not delete the post: "+err.Error(), true)
			return
		}
		log.Printf("user %s closed and deleted suggestion post %s", userID, channelID)
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

func sendErrorMessage(s *discordgo.Session, channelID, title, description string) {
	if _, err := s.ChannelMessageSendEmbed(channelID, forumdiscord.ErrorEmbed(title, description)); err != nil {
		log.Printf("send error embed: %v", err)
	}
}

func sendSuggestionWelcomeMessage(s *discordgo.Session, thread *discordgo.ThreadCreate) {
	opMention := ""
	if thread.OwnerID != "" {
		opMention = "<@" + thread.OwnerID + ">"
	}
	botName := "ArchiveTune Bot"
	botAvatar := ""
	if s.State != nil && s.State.User != nil {
		botName = s.State.User.Username
		botAvatar = s.State.User.AvatarURL("")
	}
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    botName,
			IconURL: botAvatar,
		},
		Title: "ArchiveTune Suggestion System",
		Color: 0xFEE75C,
		Description: "👋 Thank you for your suggestion!\n\n" +
			"Before our team reviews your idea, please confirm that:\n\n" +
			"🎯 **Unique Feature** — It does not already exist in ArchiveTune.\n" +
			"🔍 **No Duplicates** — It has not been suggested by another user.\n" +
			"🐛 **Not a Bug Report** — Bugs should be reported under #issues (posting bug reports here may result in a warning).\n\n" +
			"⏳ Once verified, your feedback will be reviewed by developers and contributors.",
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Need to cancel? Run /close to delete this thread.",
			IconURL: botAvatar,
		},
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Close Suggestion",
				Style:    discordgo.DangerButton,
				CustomID: "close_suggestion",
			},
		}},
	}
	if _, err := s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
		Content:    opMention,
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}); err != nil {
		log.Printf("send suggestion welcome message to %s: %v", thread.ID, err)
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

func handleCloseSuggestionButton(s *discordgo.Session, interaction *discordgo.InteractionCreate, cfg *config.Config) {
	channel, err := s.Channel(interaction.ChannelID)
	if err != nil {
		respond(s, interaction, "Could not fetch post info: "+err.Error(), true)
		return
	}
	if channel.Type != discordgo.ChannelTypeGuildPublicThread && channel.Type != discordgo.ChannelTypeGuildPrivateThread {
		respond(s, interaction, "This button can only be used inside a suggestion post.", true)
		return
	}
	if channel.ParentID != forumdiscord.SuggestionChannelID {
		respond(s, interaction, "This button can only be used in the suggestion channel.", true)
		return
	}
	userID := requesterID(interaction)
	if channel.OwnerID != userID {
		respond(s, interaction, "Only the post creator can use this button.", true)
		return
	}
	if _, err := s.ChannelDelete(interaction.ChannelID); err != nil {
		respond(s, interaction, "Could not delete the post: "+err.Error(), true)
		return
	}
	log.Printf("user %s closed and deleted suggestion post %s via button", userID, interaction.ChannelID)
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
			componentError(s, i, "This media selection expired or belongs to another user. Run `/ytd` again.")
			return
		}
		updated := mediaInfoEmbed(selection.Info, fmt.Sprintf("Selected quality: `%s`. Press **Download** to start, or **Cancel** to discard this request.", displayQuality(selection.Quality)), selection.Request.Type)
		respondComponentUpdate(s, i, updated, ytdComponents(selectionID, selection))
	case "download":
		selection, ok := selections.Take(selectionID, userID)
		if !ok {
			componentError(s, i, "Choose a quality first, or run `/ytd` again because this selection expired.")
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
