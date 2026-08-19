package discord

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestOldestArchiveTimestamp(t *testing.T) {
	oldest := time.Date(2026, time.August, 1, 10, 0, 0, 0, time.UTC)
	newer := oldest.Add(2 * time.Hour)
	got, err := oldestArchiveTimestamp([]*discordgo.Channel{
		{ID: "1", ThreadMetadata: &discordgo.ThreadMetadata{ArchiveTimestamp: newer}},
		{ID: "2", ThreadMetadata: &discordgo.ThreadMetadata{ArchiveTimestamp: oldest}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(oldest) {
		t.Fatalf("expected %v, got %v", oldest, got)
	}
}

func TestOldestArchiveTimestampRejectsMissingTimestamps(t *testing.T) {
	if _, err := oldestArchiveTimestamp([]*discordgo.Channel{{ID: "1"}}); err == nil {
		t.Fatal("expected missing timestamps to be rejected")
	}
}
