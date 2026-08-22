package media

import "testing"

func TestVideoFormatAddsBestAudioForVideoOnlyID(t *testing.T) {
	info := Info{Formats: []Format{{ID: "137", VCodec: "avc1", ACodec: "none"}}}
	if got := videoFormat("137", info); got != "137+bestaudio/best" {
		t.Fatalf("expected video-only format to add audio, got %q", got)
	}
}

func TestVideoFormatKeepsCombinedFormat(t *testing.T) {
	info := Info{Formats: []Format{{ID: "18", VCodec: "avc1", ACodec: "mp4a"}}}
	if got := videoFormat("18", info); got != "18" {
		t.Fatalf("expected combined format to remain unchanged, got %q", got)
	}
}

func TestFormatSummaryFiltersMediaType(t *testing.T) {
	info := Info{Formats: []Format{
		{ID: "137", Ext: "mp4", Width: 1920, Height: 1080, VCodec: "avc1", ACodec: "none"},
		{ID: "251", Ext: "webm", ACodec: "opus", VCodec: "none", Bitrate: 160},
	}}
	videoSummary := FormatSummary(info, MediaVideo)
	if len(videoSummary) == 0 || !contains(videoSummary, "137") || contains(videoSummary, "251") {
		t.Fatalf("unexpected video summary: %q", videoSummary)
	}
	audioSummary := FormatSummary(info, MediaAudio)
	if !contains(audioSummary, "251") || contains(audioSummary, "137") {
		t.Fatalf("unexpected audio summary: %q", audioSummary)
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
