package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type SuggestionStatusAction struct {
	Command     string
	TagName     string
	TitlePrefix string
}

var suggestionStatusActions = map[string]SuggestionStatusAction{
	".dupe":        {Command: ".dupe", TagName: "Duplicate", TitlePrefix: "[DUPLICATE]"},
	".done":        {Command: ".done", TagName: "Done", TitlePrefix: "[DONE]"},
	".in-progress": {Command: ".in-progress", TagName: "In Progress...", TitlePrefix: "[IN PROGRESS]"},
	".exist":       {Command: ".exist", TagName: "Already exist", TitlePrefix: "[ALREADY EXIST]"},
}

func ParseSuggestionStatusCommand(content string) (action SuggestionStatusAction, link string, matched, valid bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) == 0 {
		return SuggestionStatusAction{}, "", false, false
	}
	action, matched = suggestionStatusActions[strings.ToLower(fields[0])]
	if !matched {
		return SuggestionStatusAction{}, "", false, false
	}
	if len(fields) != 2 {
		return action, "", true, false
	}
	return action, fields[1], true, true
}

func (m *Manager) ApplySuggestionStatusAction(threadID string, action SuggestionStatusAction) (*discordgo.Channel, error) {
	thread, cfg, err := m.ManagedThread(threadID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(cfg.Name), "suggestion") {
		return nil, fmt.Errorf("%s chỉ được phép dùng cho Forum Channel suggestion", action.Command)
	}

	tagID, err := m.tagID(cfg, action.TagName)
	if err != nil {
		return nil, err
	}
	baseName := strings.TrimSpace(statusPrefix.ReplaceAllString(thread.Name, ""))
	if baseName == "" {
		baseName = thread.Name
	}
	newName := strings.TrimSpace(action.TitlePrefix + " " + baseName)
	archived, locked := true, true
	appliedTags := []string{tagID}
	return m.editThread(threadID, &discordgo.ChannelEdit{
		Name:        newName,
		Archived:    &archived,
		Locked:      &locked,
		AppliedTags: &appliedTags,
	})
}

func (m *Manager) ThreadAuthorMention(threadID string) (string, error) {
	thread, err := m.session.Channel(threadID)
	if err != nil {
		return "", err
	}
	if thread.OwnerID == "" {
		return "", nil
	}
	return "<@" + thread.OwnerID + ">", nil
}
