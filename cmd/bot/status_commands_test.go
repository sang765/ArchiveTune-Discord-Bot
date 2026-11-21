package main

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestStatusSlashCommandsCoverEveryWorkflow(t *testing.T) {
	expected := []string{
		"accept", "dupe", "reject", "done", "in-progress", "exist",
		"solved", "false", "false-report", "accepted", "rejected", "maybe",
		"duplicate", "already-exist", "tba", "tbd", "problem", "question",
		"stable", "nightly", "meta",
	}
	if len(statusCommandSpecs) != len(expected) {
		t.Fatalf("expected %d status workflows, got %d", len(expected), len(statusCommandSpecs))
	}
	for _, name := range expected {
		if _, ok := statusActionByName(name); !ok {
			t.Errorf("missing status slash command /%s", name)
		}
	}

	generated := statusSlashCommands()
	if len(generated) != len(expected) {
		t.Fatalf("expected %d generated commands, got %d", len(expected), len(generated))
	}
	seen := make(map[string]struct{}, len(generated))
	for _, command := range generated {
		if command.Name == "" || len(command.Name) > 32 {
			t.Errorf("invalid slash command name %q", command.Name)
		}
		if command.Description == "" {
			t.Errorf("command /%s has no description", command.Name)
		}
		if _, duplicate := seen[command.Name]; duplicate {
			t.Errorf("duplicate slash command /%s", command.Name)
		}
		seen[command.Name] = struct{}{}

		options := map[string]*discordgo.ApplicationCommandOption{}
		for _, option := range command.Options {
			options[option.Name] = option
		}
		if postID, ok := options["post_id"]; !ok || postID.Required {
			t.Errorf("/%s must have an optional post_id option", command.Name)
		}
		switch command.Name {
		case "dupe":
			if postLink, ok := options["post_link"]; !ok || !postLink.Required {
				t.Error("/dupe must have a required post_link option")
			}
		case "reject":
			if reason, ok := options["reason"]; !ok || !reason.Required {
				t.Error("/reject must have a required reason option")
			}
		default:
			if _, ok := options["post_link"]; ok {
				t.Errorf("/%s should not have a post_link option", command.Name)
			}
			if _, ok := options["reason"]; ok {
				t.Errorf("/%s should not have a reason option", command.Name)
			}
		}
	}
}

func TestStatusSlashCommandsDoNotShadowBaseCommands(t *testing.T) {
	baseNames := map[string]struct{}{
		"help": {}, "ytd": {}, "forum-sync": {}, "fix-suggestion": {},
		"tag-add": {}, "tag-remove": {}, "post-rename": {}, "post-state": {}, "close": {},
	}
	for _, command := range statusSlashCommands() {
		if _, exists := baseNames[command.Name]; exists {
			t.Errorf("/%s collides with an existing bot command", command.Name)
		}
	}
}
