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
		Name:        "forum-sync",
		Description: "Đồng bộ guideline, tag và trạng thái bắt buộc tag cho Forum Channel",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "channel", Description: "Forum Channel cần đồng bộ", Type: discordgo.ApplicationCommandOptionChannel, Required: true},
		},
	},
	{
		Name:        "fix-suggestion",
		Description: "Gắn tag Maybe cho các suggestion post đang chưa có tag",
	},
	{
		Name:        "tag-add",
		Description: "Thêm một tag vào post hiện tại hoặc post được chỉ định",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "tag", Description: "Tên tag đã cấu hình", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "post_id", Description: "ID post; bỏ trống để dùng post hiện tại", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
	{
		Name:        "tag-remove",
		Description: "Gỡ một tag khỏi post hiện tại hoặc post được chỉ định",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "tag", Description: "Tên tag đã cấu hình", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "post_id", Description: "ID post; bỏ trống để dùng post hiện tại", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
	{
		Name:        "post-rename",
		Description: "Đổi tên một post trong Forum Channel được quản lý",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "name", Description: "Tên mới của post", Type: discordgo.ApplicationCommandOptionString, Required: true},
			{Name: "post_id", Description: "ID post; bỏ trống để dùng post hiện tại", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	},
	{
		Name:        "post-state",
		Description: "Đóng/mở hoặc khóa/mở khóa post",
		Options: []*discordgo.ApplicationCommandOption{
			{Name: "state", Description: "Trạng thái muốn đặt", Type: discordgo.ApplicationCommandOptionString, Required: true, Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "open", Value: "open"},
				{Name: "close", Value: "close"},
				{Name: "lock", Value: "lock"},
				{Name: "unlock", Value: "unlock"},
			}},
			{Name: "post_id", Description: "ID post; bỏ trống để dùng post hiện tại", Type: discordgo.ApplicationCommandOptionString, Required: false},
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
		if !forumdiscord.HasModeratorAccess(interaction, cfg) {
			respond(s, interaction, "Bạn cần quyền `Manage Threads`, `Administrator` hoặc role moderator được cấu hình để dùng lệnh này.", true)
			return
		}
		handleCommand(s, interaction, manager, cfg)
	})
	session.AddHandler(func(s *discordgo.Session, message *discordgo.MessageCreate) {
		if message.Author == nil || message.Author.Bot || message.GuildID != cfg.GuildID {
			return
		}
		if statusAction, link, matched, valid := forumdiscord.ParseSuggestionStatusCommand(message.Content); matched {
			if !valid {
				if statusAction.Command == ".dupe" {
					_, _ = s.ChannelMessageSend(message.ChannelID, "Cú pháp: `.dupe <Discord post link hoặc message link>`")
				} else {
					_, _ = s.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Cú pháp: `%s` — hãy gửi lệnh trực tiếp trong post cần xử lý.", statusAction.Command))
				}
				return
			}
			if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
				_, _ = s.ChannelMessageSend(message.ChannelID, "Bạn cần quyền `Manage Threads`, `Administrator` hoặc role moderator được cấu hình để dùng suggestion status command.")
				return
			}
			targetID := message.ChannelID
			if link != "" {
				var err error
				targetID, err = forumdiscord.ParseDiscordPostLink(link, cfg.GuildID)
				if err != nil {
					_, _ = s.ChannelMessageSend(message.ChannelID, "Không thể đọc post link: "+err.Error())
					return
				}
			}
			authorMention, authorErr := manager.ThreadAuthorMention(targetID)
			if authorErr != nil {
				log.Printf("get author for %s: %v", targetID, authorErr)
			}
			updated, err := manager.ApplySuggestionStatusAction(targetID, statusAction)
			if err != nil {
				_, _ = s.ChannelMessageSend(message.ChannelID, "Không thể xử lý "+statusAction.Command+": "+err.Error())
				return
			}
			messageText := fmt.Sprintf("Đã áp dụng `%s`: thay toàn bộ tag bằng `%s`, đóng, khóa và đổi tên thành **%s**.", statusAction.Command, statusAction.TagName, updated.Name)
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
			return
		}
		if !forumdiscord.HasModeratorMessageAccess(message, cfg) {
			_, _ = s.ChannelMessageSend(message.ChannelID, "Bạn cần quyền `Manage Threads`, `Administrator` hoặc role moderator được cấu hình để dùng prefix command.")
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
		updated, err := manager.ApplyPrefixAction(message.ChannelID, action)
		if err != nil {
			_, _ = s.ChannelMessageSend(message.ChannelID, "Không thể xử lý "+action.Command+": "+err.Error())
			return
		}
		messageText := fmt.Sprintf("Đã áp dụng `%s`: tag `%s`, khóa post và đổi tên thành **%s**.", action.Command, action.TagName, updated.Name)
		if action.Command == ".tba" || action.Command == ".tbd" {
			messageText = fmt.Sprintf("Đã áp dụng `%s`: xóa tag cũ và chỉ giữ tag `%s`; post không bị lock, close hoặc đổi tên.", action.Command, action.TagName)
		}
		if authorMention != "" {
			messageText = authorMention + " " + messageText
		}
		if originalCommand != "" {
			messageText = fmt.Sprintf("Mình đã tự sửa `%s` thành `%s`. %s", originalCommand, action.Command, messageText)
		}
		_, _ = s.ChannelMessageSend(message.ChannelID, messageText)
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

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate, manager *forumdiscord.Manager, cfg *config.Config) {
	data := i.ApplicationCommandData()
	switch data.Name {
	case "forum-sync":
		channelID := optionString(data.Options, "channel")
		channel, err := manager.SyncChannel(channelID)
		if err != nil {
			respond(s, i, err.Error(), true)
			return
		}
		respond(s, i, fmt.Sprintf("Đã đồng bộ Forum Channel %s với %d tag.", channel.Mention(), len(channel.AvailableTags)), false)
	case "fix-suggestion":
		scanned, fixed, err := manager.FixSuggestion()
		if err != nil {
			respond(s, i, err.Error(), true)
			return
		}
		respond(s, i, fmt.Sprintf("Đã quét %d suggestion post và gắn tag `Maybe` cho %d post chưa có tag.", scanned, fixed), false)
	case "tag-add":
		postID := postIDFrom(i, data.Options)
		updated, err := manager.ApplyTag(postID, optionString(data.Options, "tag"))
		if err != nil {
			respond(s, i, err.Error(), true)
			return
		}
		respond(s, i, fmt.Sprintf("Đã thêm tag vào post %s.", updated.Mention()), false)
	case "tag-remove":
		postID := postIDFrom(i, data.Options)
		updated, err := manager.RemoveTag(postID, optionString(data.Options, "tag"))
		if err != nil {
			respond(s, i, err.Error(), true)
			return
		}
		respond(s, i, fmt.Sprintf("Đã gỡ tag khỏi post %s.", updated.Mention()), false)
	case "post-rename":
		postID := postIDFrom(i, data.Options)
		updated, err := manager.RenameThread(postID, optionString(data.Options, "name"))
		if err != nil {
			respond(s, i, err.Error(), true)
			return
		}
		respond(s, i, fmt.Sprintf("Đã đổi tên post thành **%s**.", updated.Name), false)
	case "post-state":
		postID := postIDFrom(i, data.Options)
		state := optionString(data.Options, "state")
		if state != "open" && state != "close" && state != "lock" && state != "unlock" {
			respond(s, i, "Trạng thái không hợp lệ.", true)
			return
		}
		updated, err := manager.SetThreadState(postID, state)
		if err != nil {
			respond(s, i, err.Error(), true)
			return
		}
		respond(s, i, fmt.Sprintf("Đã đặt trạng thái `%s` cho post %s.", state, updated.Mention()), false)
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
