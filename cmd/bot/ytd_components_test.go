package main

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/sang765/discord-forum-bot/internal/media"
)

func TestYTDComponentsRequireQualityBeforeDownload(t *testing.T) {
	selection := media.Selection{
		Request: media.Request{Type: media.MediaAudio},
		Info: media.Info{Formats: []media.Format{
			{ID: "251", Ext: "webm", ACodec: "opus", VCodec: "none", Bitrate: 134},
		}},
	}
	components := ytdComponents("selection", selection)
	if len(components) != 2 {
		t.Fatalf("expected two component rows, got %d", len(components))
	}
	buttons := components[1].(discordgo.ActionsRow).Components
	download := buttons[0].(discordgo.Button)
	if !download.Disabled {
		t.Fatal("expected download button to be disabled before quality selection")
	}
	selection.Quality = "251"
	buttons = ytdComponents("selection", selection)[1].(discordgo.ActionsRow).Components
	download = buttons[0].(discordgo.Button)
	if download.Disabled {
		t.Fatal("expected download button to be enabled after quality selection")
	}
}
