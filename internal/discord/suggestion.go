package discord

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (m *Manager) MaybeTagIfMissing(threadID string) (bool, error) {
	thread, cfg, err := m.ManagedThread(threadID)
	if err != nil {
		return false, err
	}
	if cfg.ID != SuggestionChannelID {
		return false, nil
	}
	if len(thread.AppliedTags) > 0 {
		return false, nil
	}

	tagID, err := m.tagID(cfg, "Maybe")
	if err != nil {
		return false, err
	}
	appliedTags := []string{tagID}
	if _, err := m.editThread(threadID, &discordgo.ChannelEdit{AppliedTags: &appliedTags}); err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) FixSuggestion() (scanned, fixed int, err error) {
	cfg, ok := m.config.Channel(SuggestionChannelID)
	if !ok {
		return 0, 0, fmt.Errorf("suggestion channel %s is not configured", SuggestionChannelID)
	}

	active, err := m.session.ThreadsActive(cfg.ID)
	if err != nil {
		return 0, 0, fmt.Errorf("list active suggestion posts: %w", err)
	}
	scanned, fixed, err = m.fixThreadList(active.Threads, scanned, fixed)
	if err != nil {
		return scanned, fixed, err
	}

	var before *time.Time
	for {
		archived, listErr := m.session.ThreadsArchived(cfg.ID, before, 100)
		if listErr != nil {
			return scanned, fixed, fmt.Errorf("list archived suggestion posts: %w", listErr)
		}
		scanned, fixed, err = m.fixThreadList(archived.Threads, scanned, fixed)
		if err != nil {
			return scanned, fixed, err
		}
		if !archived.HasMore || len(archived.Threads) == 0 {
			break
		}

		oldest, parseErr := oldestArchiveTimestamp(archived.Threads)
		if parseErr != nil {
			return scanned, fixed, parseErr
		}
		before = &oldest
	}
	return scanned, fixed, nil
}

func (m *Manager) fixThreadList(threads []*discordgo.Channel, scanned, fixed int) (int, int, error) {
	for _, thread := range threads {
		if thread == nil || thread.ParentID != SuggestionChannelID {
			continue
		}
		scanned++
		changed, err := m.MaybeTagIfMissing(thread.ID)
		if err != nil {
			return scanned, fixed, fmt.Errorf("fix suggestion post %s: %w", thread.ID, err)
		}
		if changed {
			fixed++
		}
	}
	return scanned, fixed, nil
}

func oldestArchiveTimestamp(threads []*discordgo.Channel) (time.Time, error) {
	var oldest time.Time
	for _, thread := range threads {
		if thread == nil || thread.ThreadMetadata == nil || thread.ThreadMetadata.ArchiveTimestamp.IsZero() {
			continue
		}
		timestamp := thread.ThreadMetadata.ArchiveTimestamp
		if oldest.IsZero() || timestamp.Before(oldest) {
			oldest = timestamp
		}
	}
	if oldest.IsZero() {
		return time.Time{}, fmt.Errorf("archived posts did not include archive timestamps")
	}
	return oldest, nil
}
