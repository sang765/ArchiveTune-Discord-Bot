package main

import (
	"strings"
	"testing"
)

func TestWorkflowDescriptionTemplates(t *testing.T) {
	falseDescription := workflowDescription(".false", "[FALSE REPORT] Bad issue", "ignored", "", nil)
	if !strings.Contains(falseDescription, "You may be warned, muted, kicked, or even worse, banned") {
		t.Fatalf("false report warning is missing: %q", falseDescription)
	}

	duplicateDescription := workflowDescription(".dupe", "[DUPLICATE] Current suggestion", "ignored", "", &duplicateReferenceData{
		name:          "Original suggestion",
		link:          "https://discord.com/channels/123/456",
		authorMention: "<@789>",
	})
	for _, expected := range []string{
		"**What suggestion post has been duplicated?**",
		"**[\"Original suggestion\"](https://discord.com/channels/123/456)** by <@789>",
	} {
		if !strings.Contains(duplicateDescription, expected) {
			t.Fatalf("duplicate description is missing %q: %q", expected, duplicateDescription)
		}
	}

	rejectDescription := workflowDescription(".reject", "[REJECTED] Suggestion", "ignored", "Not feasible", nil)
	if !strings.Contains(rejectDescription, "```text\nNot feasible\n```") {
		t.Fatalf("reject reason text block is missing: %q", rejectDescription)
	}

	safeRejectDescription := workflowDescription(".reject", "[REJECTED] Suggestion", "ignored", "<@123> @everyone", nil)
	if strings.Contains(safeRejectDescription, "<@123>") || strings.Contains(safeRejectDescription, "@everyone") {
		t.Fatalf("reject reason still contains a mention: %q", safeRejectDescription)
	}
	if !strings.Contains(safeRejectDescription, "<@\u200b123> @\u200beveryone") {
		t.Fatalf("reject reason was not safely escaped: %q", safeRejectDescription)
	}
}
