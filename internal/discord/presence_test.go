package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestArchiveTunePresence(t *testing.T) {
	presence := ArchiveTunePresence()
	if presence.Status != string(discordgo.StatusDoNotDisturb) {
		t.Fatalf("expected DND status, got %q", presence.Status)
	}
	if len(presence.Activities) != 1 {
		t.Fatalf("expected one activity, got %d", len(presence.Activities))
	}
	activity := presence.Activities[0]
	if activity.Type != discordgo.ActivityTypeCustom {
		t.Fatalf("expected custom activity type, got %d", activity.Type)
	}
	if activity.State != ArchiveTuneCustomStatus {
		t.Fatalf("expected custom status %q, got %q", ArchiveTuneCustomStatus, activity.State)
	}
}
