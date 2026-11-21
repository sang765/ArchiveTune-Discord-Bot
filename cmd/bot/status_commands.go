package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/config"
	forumdiscord "github.com/sang765/discord-forum-bot/internal/discord"
)

// statusKind selects the underlying workflow implementation. Suggestion actions
// close, lock and replace all tags; prefix actions only apply a tag, lock and
// rename (except .tba/.tbd, which only replace tags).
type statusKind int

const (
	statusKindSuggestion statusKind = iota
	statusKindPrefix
)

type statusCommandSpec struct {
	name        string
	command     string
	description string
	kind        statusKind
	needsReason bool
	needsLink   bool
	workflow    bool
}

// statusCommandSpecs preserves every prefix workflow as a slash command. The
// command names intentionally match the old prefix names without the dot so
// existing moderators keep the same muscle memory.
var statusCommandSpecs = []statusCommandSpec{
	{name: "accept", command: ".accept", description: "Mark a suggestion as accepted (close, lock, rename)", kind: statusKindSuggestion, workflow: true},
	{name: "dupe", command: ".dupe", description: "Mark a suggestion as a duplicate of another post", kind: statusKindSuggestion, needsLink: true, workflow: true},
	{name: "reject", command: ".reject", description: "Reject a suggestion with a reason", kind: statusKindSuggestion, needsReason: true, workflow: true},
	{name: "done", command: ".done", description: "Mark a suggestion as done", kind: statusKindSuggestion, workflow: true},
	{name: "in-progress", command: ".in-progress", description: "Mark a suggestion as in progress", kind: statusKindSuggestion, workflow: true},
	{name: "exist", command: ".exist", description: "Mark a suggestion as already existing", kind: statusKindSuggestion, workflow: true},

	{name: "solved", command: ".solved", description: "Mark an issue as solved (issues channel only)", kind: statusKindPrefix, workflow: true},
	{name: "false", command: ".false", description: "Mark an issue as a false report (issues channel only)", kind: statusKindPrefix, workflow: true},
	{name: "false-report", command: ".false-report", description: "Alias of /false", kind: statusKindPrefix, workflow: true},

	{name: "accepted", command: ".accepted", description: "Alias of /accept", kind: statusKindPrefix, workflow: true},
	{name: "rejected", command: ".rejected", description: "Alias of /reject", kind: statusKindPrefix, workflow: true},
	{name: "maybe", command: ".maybe", description: "Apply the Maybe tag and lock the post", kind: statusKindPrefix},
	{name: "duplicate", command: ".duplicate", description: "Apply the Duplicate tag and lock the post", kind: statusKindPrefix},
	{name: "already-exist", command: ".already-exist", description: "Apply the Already exist tag and lock the post", kind: statusKindPrefix},
	{name: "tba", command: ".tba", description: "Keep only the TBA tag without locking", kind: statusKindPrefix},
	{name: "tbd", command: ".tbd", description: "Keep only the TBD tag without locking", kind: statusKindPrefix},
	{name: "problem", command: ".problem", description: "Apply the Problem tag and lock the post", kind: statusKindPrefix},
	{name: "question", command: ".question", description: "Apply the Question tag and lock the post", kind: statusKindPrefix},
	{name: "stable", command: ".stable", description: "Apply the Stable Version tag and lock the post", kind: statusKindPrefix},
	{name: "nightly", command: ".nightly", description: "Apply the Nightly Version tag and lock the post", kind: statusKindPrefix},
	{name: "meta", command: ".meta", description: "Apply the meta tag and lock the post", kind: statusKindPrefix},
}

func statusSlashCommands() []*discordgo.ApplicationCommand {
	commands := make([]*discordgo.ApplicationCommand, 0, len(statusCommandSpecs))
	for _, spec := range statusCommandSpecs {
		options := make([]*discordgo.ApplicationCommandOption, 0, 3)
		if spec.needsLink {
			options = append(options, &discordgo.ApplicationCommandOption{
				Name:        "post_link",
				Description: "Discord post or message link that this post duplicates",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			})
		}
		if spec.needsReason {
			options = append(options, &discordgo.ApplicationCommandOption{
				Name:        "reason",
				Description: "Rejection reason (maximum 1,000 characters)",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			})
		}
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        "post_id",
			Description: "Post ID; leave empty to use the current post",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    false,
		})
		commands = append(commands, &discordgo.ApplicationCommand{
			Name:        spec.name,
			Description: spec.description,
			Options:     options,
		})
	}
	return commands
}

func statusActionByName(name string) (statusCommandSpec, bool) {
	for _, spec := range statusCommandSpecs {
		if spec.name == name {
			return spec, true
		}
	}
	return statusCommandSpec{}, false
}

func handleStatusCommand(s *discordgo.Session, i *discordgo.InteractionCreate, manager *forumdiscord.Manager, cfg *config.Config, spec statusCommandSpec) {
	data := i.ApplicationCommandData()
	postID := postIDFrom(i, data.Options)

	reason := ""
	if spec.needsReason {
		reason = optionString(data.Options, "reason")
		if reason == "" || len([]rune(reason)) > 1000 {
			editInteraction(s, i, "The reason is required and must be at most 1,000 characters.", true)
			return
		}
	}

	var duplicateReference *duplicateReferenceData
	if spec.needsLink {
		link := optionString(data.Options, "post_link")
		referencedID, err := forumdiscord.ParseDiscordPostLink(link, cfg.GuildID)
		if err != nil {
			editInteraction(s, i, "Could not parse post link: "+err.Error(), true)
			return
		}
		referencedPost, referencedCfg, err := manager.ManagedThread(referencedID)
		if err != nil {
			editInteraction(s, i, "Could not load the referenced suggestion post: "+err.Error(), true)
			return
		}
		if !strings.EqualFold(strings.TrimSpace(referencedCfg.Name), "suggestion") {
			editInteraction(s, i, "The referenced post is not in the managed suggestion Forum Channel.", true)
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

	authorMention := ""
	if spec.workflow {
		mention, authorErr := manager.ThreadAuthorMention(postID)
		if authorErr != nil {
			log.Printf("get author for %s: %v", postID, authorErr)
		}
		authorMention = mention
	}

	var updated *discordgo.Channel
	var err error
	tagName := ""
	if spec.kind == statusKindSuggestion {
		action, ok := forumdiscord.SuggestionStatusActionFor(spec.command)
		if !ok {
			editInteraction(s, i, "Unknown suggestion workflow.", true)
			return
		}
		tagName = action.TagName
		updated, err = manager.ApplySuggestionStatusAction(postID, action)
	} else {
		action, ok := forumdiscord.PrefixActionFor(spec.command)
		if !ok {
			editInteraction(s, i, "Unknown status workflow.", true)
			return
		}
		tagName = action.TagName
		updated, err = manager.ApplyPrefixAction(postID, action)
	}
	if err != nil {
		editInteraction(s, i, fmt.Sprintf("Could not process /%s: %v", spec.name, err), true)
		return
	}

	if spec.command == ".tba" || spec.command == ".tbd" {
		editInteraction(s, i, fmt.Sprintf("Applied `/%s`: removed all previous tags and kept only `%s`; the post was not locked, closed, or renamed.", spec.name, tagName), false)
		return
	}

	if spec.workflow {
		details := fmt.Sprintf("The post was tagged `%s`, closed and locked, and renamed to **%s**.", tagName, updated.Name)
		if spec.command == ".reject" {
			details = fmt.Sprintf("The post was tagged `Reject`, closed and locked, and renamed to **%s**. Reason: %s", updated.Name, reason)
		}
		sendWorkflowMessage(s, postID, i.GuildID, authorMention, spec.command, updated.Name, details, reason, duplicateReference)
	}

	editInteraction(s, i, fmt.Sprintf("Applied `/%s` to %s.", spec.name, updated.Mention()), false)
}
