package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/config"
)

const requireTagFlag discordgo.ChannelFlags = 1 << 4

// Manager contains the deterministic Discord operations of this bot.
type Manager struct {
	session *discordgo.Session
	config  *config.Config
}

func NewManager(session *discordgo.Session, cfg *config.Config) *Manager {
	return &Manager{session: session, config: cfg}
}

func (m *Manager) SyncConfiguredChannels() error {
	for _, channel := range m.config.Channels {
		if _, err := m.SyncChannel(channel.ID); err != nil {
			return fmt.Errorf("sync %s (%s): %w", channel.Name, channel.ID, err)
		}
	}
	return nil
}

func (m *Manager) SyncChannel(channelID string) (*discordgo.Channel, error) {
	cfg, ok := m.config.Channel(channelID)
	if !ok {
		return nil, fmt.Errorf("channel %s is not managed", channelID)
	}

	current, err := m.session.Channel(channelID)
	if err != nil {
		return nil, fmt.Errorf("get channel: %w", err)
	}
	if current.Type != discordgo.ChannelTypeGuildForum {
		return nil, fmt.Errorf("channel %s is type %d, expected forum", channelID, current.Type)
	}

	tags := mergeTags(current.AvailableTags, cfg.Tags, m.config.ReplaceExistingTags)
	flags := current.Flags
	if cfg.RequireTag {
		flags |= requireTagFlag
	} else {
		flags &^= requireTagFlag
	}

	edit := &discordgo.ChannelEdit{
		Topic:         cfg.PostGuidelines,
		Flags:         &flags,
		AvailableTags: &tags,
	}
	updated, err := m.session.ChannelEdit(channelID, edit)
	if err != nil {
		return nil, fmt.Errorf("edit forum channel: %w", err)
	}
	return updated, nil
}

func mergeTags(existing []discordgo.ForumTag, declared []config.TagConfig, replace bool) []discordgo.ForumTag {
	result := make([]discordgo.ForumTag, 0, len(existing)+len(declared))
	byName := make(map[string]int, len(existing)+len(declared))
	for _, tag := range existing {
		key := strings.ToLower(strings.TrimSpace(tag.Name))
		if key == "" {
			continue
		}
		byName[key] = len(result)
		result = append(result, tag)
	}

	if replace {
		result = result[:0]
		byName = make(map[string]int, len(declared))
	}
	for _, declaredTag := range declared {
		name := strings.TrimSpace(declaredTag.Name)
		key := strings.ToLower(name)
		newTag := discordgo.ForumTag{Name: name, EmojiName: declaredTag.Emoji}
		if index, ok := byName[key]; ok {
			newTag.ID = result[index].ID
			newTag.Moderated = result[index].Moderated
			result[index] = newTag
			continue
		}
		byName[key] = len(result)
		result = append(result, newTag)
	}
	return result
}

func (m *Manager) ManagedThread(threadID string) (*discordgo.Channel, config.ChannelConfig, error) {
	thread, err := m.session.Channel(threadID)
	if err != nil {
		return nil, config.ChannelConfig{}, fmt.Errorf("get post: %w", err)
	}
	parentID := thread.ParentID
	if parentID == "" {
		return nil, config.ChannelConfig{}, fmt.Errorf("%s is not a forum post", threadID)
	}
	cfg, ok := m.config.Channel(parentID)
	if !ok {
		return nil, config.ChannelConfig{}, fmt.Errorf("post %s is not in a managed forum", threadID)
	}
	return thread, cfg, nil
}

func (m *Manager) ApplyTag(threadID, tagName string) (*discordgo.Channel, error) {
	thread, cfg, err := m.ManagedThread(threadID)
	if err != nil {
		return nil, err
	}
	tagID, err := m.tagID(cfg, tagName)
	if err != nil {
		return nil, err
	}
	for _, applied := range thread.AppliedTags {
		if applied == tagID {
			return thread, nil
		}
	}
	if len(thread.AppliedTags) >= 5 {
		return nil, fmt.Errorf("post already has the maximum of 5 applied tags")
	}
	applied := append([]string(nil), thread.AppliedTags...)
	applied = append(applied, tagID)
	return m.editThread(threadID, &discordgo.ChannelEdit{AppliedTags: &applied})
}

func (m *Manager) RemoveTag(threadID, tagName string) (*discordgo.Channel, error) {
	thread, cfg, err := m.ManagedThread(threadID)
	if err != nil {
		return nil, err
	}
	tagID, err := m.tagID(cfg, tagName)
	if err != nil {
		return nil, err
	}
	applied := make([]string, 0, len(thread.AppliedTags))
	for _, appliedTag := range thread.AppliedTags {
		if appliedTag != tagID {
			applied = append(applied, appliedTag)
		}
	}
	return m.editThread(threadID, &discordgo.ChannelEdit{AppliedTags: &applied})
}

func (m *Manager) RenameThread(threadID, name string) (*discordgo.Channel, error) {
	if strings.TrimSpace(name) == "" || len([]rune(name)) > 100 {
		return nil, fmt.Errorf("post name must contain 1-100 characters")
	}
	if _, _, err := m.ManagedThread(threadID); err != nil {
		return nil, err
	}
	return m.editThread(threadID, &discordgo.ChannelEdit{Name: strings.TrimSpace(name)})
}

func (m *Manager) SetThreadState(threadID, state string) (*discordgo.Channel, error) {
	if _, _, err := m.ManagedThread(threadID); err != nil {
		return nil, err
	}
	edit := &discordgo.ChannelEdit{}
	switch state {
	case "open":
		archived := false
		edit.Archived = &archived
	case "close":
		archived := true
		edit.Archived = &archived
	case "lock":
		locked := true
		edit.Locked = &locked
	case "unlock":
		locked := false
		edit.Locked = &locked
	default:
		return nil, fmt.Errorf("unknown post state %q", state)
	}
	return m.editThread(threadID, edit)
}

func (m *Manager) tagID(cfg config.ChannelConfig, tagName string) (string, error) {
	needle := strings.ToLower(strings.TrimSpace(tagName))
	for _, declared := range cfg.Tags {
		if strings.ToLower(strings.TrimSpace(declared.Name)) != needle {
			continue
		}
		channel, err := m.session.Channel(cfg.ID)
		if err != nil {
			return "", fmt.Errorf("get forum tags: %w", err)
		}
		for _, available := range channel.AvailableTags {
			if strings.EqualFold(available.Name, declared.Name) && available.ID != "" {
				return available.ID, nil
			}
		}
		return "", fmt.Errorf("tag %q is declared but has no Discord id; run /forum-sync first", declared.Name)
	}
	return "", fmt.Errorf("tag %q is not configured for #%s", tagName, cfg.Name)
}

func (m *Manager) editThread(threadID string, edit *discordgo.ChannelEdit) (*discordgo.Channel, error) {
	updated, err := m.session.ChannelEdit(threadID, edit)
	if err != nil {
		return nil, fmt.Errorf("edit post: %w", err)
	}
	return updated, nil
}

func HasModeratorAccess(i *discordgo.InteractionCreate, cfg *config.Config) bool {
	if i.Member == nil {
		return false
	}
	permissions := uint64(i.Member.Permissions)
	if permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageThreads) != 0 {
		return true
	}
	allowed := make(map[string]struct{}, len(cfg.ModeratorRoleIDs))
	for _, roleID := range cfg.ModeratorRoleIDs {
		allowed[roleID] = struct{}{}
	}
	for _, roleID := range i.Member.Roles {
		if _, ok := allowed[roleID]; ok {
			return true
		}
	}
	return false
}
