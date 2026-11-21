package discord

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PrefixAction struct {
	Command     string
	TagName     string
	TitlePrefix string
}

var prefixActions = map[string]PrefixAction{
	".solved":        {Command: ".solved", TagName: "Solved", TitlePrefix: "[SOLVED]"},
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
	".false":         {Command: ".false", TagName: "False report", TitlePrefix: "[FALSE REPORT]"},
	".false-report":  {Command: ".false-report", TagName: "False report", TitlePrefix: "[FALSE REPORT]"},
	".meta":          {Command: ".meta", TagName: "meta", TitlePrefix: "[META]"},
}

var statusPrefix = regexp.MustCompile(`^\[[^\]]+\]\s*`)

// PrefixActionFor looks up a workflow action by its command name. It is shared by
// the message parser and the slash-command handlers so both use the same tag and
// title-prefix data.
func PrefixActionFor(command string) (PrefixAction, bool) {
	action, ok := prefixActions[strings.ToLower(strings.TrimSpace(command))]
	return action, ok
}

func ParsePrefixCommand(content string) (PrefixAction, bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) != 1 {
		return PrefixAction{}, false
	}
	return PrefixActionFor(fields[0])
}

// GuessPrefixCommand returns a command only when there is one unambiguous close match.
// It intentionally does not add typo spellings to prefixActions, so a typo is never
// treated as a real command when autocorrection is disabled.
func GuessPrefixCommand(content string, maxDistance int) (PrefixAction, string, bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) != 1 || maxDistance <= 0 {
		return PrefixAction{}, "", false
	}
	candidate := strings.ToLower(fields[0])
	bestDistance := maxDistance + 1
	var bestAction PrefixAction
	var bestCommand string
	tie := false
	for command, action := range prefixActions {
		distance := levenshtein(candidate, command)
		if distance > maxDistance {
			continue
		}
		if distance < bestDistance {
			bestDistance = distance
			bestAction = action
			bestCommand = command
			tie = false
		} else if distance == bestDistance {
			tie = true
		}
	}
	if bestCommand == "" || tie {
		return PrefixAction{}, "", false
	}
	return bestAction, bestCommand, true
}

func levenshtein(a, b string) int {
	left, right := []rune(a), []rune(b)
	previous := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range left {
		current := make([]int, len(right)+1)
		current[0] = i + 1
		for j, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = minInt(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}

func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func (m *Manager) ApplyPrefixAction(threadID string, action PrefixAction) (*discordgo.Channel, error) {
	thread, cfg, err := m.ManagedThread(threadID)
	if err != nil {
		return nil, err
	}
	if !PrefixActionAllowedForChannel(action, cfg.ID) {
		return nil, fmt.Errorf("%s is not allowed in this Forum Channel", action.Command)
	}
	if action.Command == ".tba" || action.Command == ".tbd" {
		tagID, err := m.tagID(cfg, action.TagName)
		if err != nil {
			return nil, err
		}
		appliedTags := []string{tagID}
		return m.editThread(threadID, &discordgo.ChannelEdit{AppliedTags: &appliedTags})
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
