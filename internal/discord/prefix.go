package discord

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/config"
)

type PrefixAction struct {
	Command     string
	TagName     string
	TitlePrefix string
}

var prefixActions = map[string]PrefixAction{
	".solved":        {Command: ".solved", TagName: "Solved", TitlePrefix: "[SOLVED]"},
	".sloved":        {Command: ".sloved", TagName: "Solved", TitlePrefix: "[SOLVED]"}, // common typo kept as alias
	".accept":        {Command: ".accept", TagName: "Accept", TitlePrefix: "[ACCEPTED]"},
	".accepted":      {Command: ".accepted", TagName: "Accept", TitlePrefix: "[ACCEPTED]"},
	".reject":        {Command: ".reject", TagName: "Reject", TitlePrefix: "[REJECTED]"},
	".rejected":      {Command: ".rejected", TagName: "Reject", TitlePrefix: "[REJECTED]"},
	".done":          {Command: ".done", TagName: "Done", TitlePrefix: "[DONE]"},
	".in-progress":   {Command: ".in-progress", TagName: "In Progress...", TitlePrefix: "[IN PROGRESS]"},
	".maybe":         {Command: ".maybe", TagName: "Maybe", TitlePrefix: "[MAYBE]"},
	".duplicate":     {Command: ".duplicate", TagName: "Duplicate", TitlePrefix: "[DUPLICATE]"},
	".already-exist": {Command: ".already-exist", TagName: "Already exist", TitlePrefix: "[ALREADY EXIST]"},
	".tba":           {Command: ".tba", TagName: "TBA", TitlePrefix: "[TBA]"},
	".tbd":           {Command: ".tbd", TagName: "TBD", TitlePrefix: "[TBD]"},
	".problem":       {Command: ".problem", TagName: "Problem", TitlePrefix: "[PROBLEM]"},
	".question":      {Command: ".question", TagName: "Question", TitlePrefix: "[QUESTION]"},
	".stable":        {Command: ".stable", TagName: "Stable Version", TitlePrefix: "[STABLE]"},
	".nightly":       {Command: ".nightly", TagName: "Nightly Version", TitlePrefix: "[NIGHTLY]"},
	".false-report":  {Command: ".false-report", TagName: "False report", TitlePrefix: "[FALSE REPORT]"},
	".meta":          {Command: ".meta", TagName: "meta", TitlePrefix: "[META]"},
}

var statusPrefix = regexp.MustCompile(`^\[[^\]]+\]\s*`)

func ParsePrefixCommand(content string) (PrefixAction, bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) != 1 {
		return PrefixAction{}, false
	}
	action, ok := prefixActions[strings.ToLower(fields[0])]
	return action, ok
}

func HasModeratorMessageAccess(message *discordgo.MessageCreate, cfg *config.Config) bool {
	if message == nil || message.Member == nil {
		return false
	}
	permissions := uint64(message.Member.Permissions)
	if permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageThreads) != 0 {
		return true
	}
	allowed := make(map[string]struct{}, len(cfg.ModeratorRoleIDs))
	for _, roleID := range cfg.ModeratorRoleIDs {
		allowed[roleID] = struct{}{}
	}
	for _, roleID := range message.Member.Roles {
		if _, ok := allowed[roleID]; ok {
			return true
		}
	}
	return false
}

func (m *Manager) ApplyPrefixAction(threadID string, action PrefixAction) (*discordgo.Channel, error) {
	thread, _, err := m.ManagedThread(threadID)
	if err != nil {
		return nil, err
	}
	if _, err := m.ApplyTag(threadID, action.TagName); err != nil {
		return nil, fmt.Errorf("apply prefix tag %q: %w", action.TagName, err)
	}
	if _, err := m.SetThreadState(threadID, "lock"); err != nil {
		return nil, fmt.Errorf("lock post after applying %q: %w", action.Command, err)
	}

	baseName := strings.TrimSpace(statusPrefix.ReplaceAllString(thread.Name, ""))
	if baseName == "" {
		baseName = thread.Name
	}
	newName := strings.TrimSpace(action.TitlePrefix + " " + baseName)
	updated, err := m.RenameThread(threadID, newName)
	if err != nil {
		return nil, fmt.Errorf("rename post after applying %q: %w", action.Command, err)
	}
	return updated, nil
}
