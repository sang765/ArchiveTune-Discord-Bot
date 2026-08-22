package media

import (
	"strings"
	"testing"
)

func TestOutputFileStemForYouTubeMusic(t *testing.T) {
	request := Request{URL: "https://music.youtube.com/watch?v=track", Type: MediaAudio}
	info := Info{Title: "Tên Bài Hát", Artist: "Tên Ca Sĩ"}
	if got, want := outputFileStem(request, info), "Tên Bài Hát - Tên Ca Sĩ"; got != want {
		t.Fatalf("outputFileStem() = %q, want %q", got, want)
	}
}

func TestOutputFileStemForYouTubeMusicFallsBackWithoutArtist(t *testing.T) {
	request := Request{URL: "https://music.youtube.com/watch?v=track", Type: MediaAudio}
	info := Info{Title: "Tên Bài Hát"}
	if got, want := outputFileStem(request, info), "Tên_Bài_Hát_audio"; got != want {
		t.Fatalf("outputFileStem() = %q, want %q", got, want)
	}
}

func TestOutputFileStemKeepsYouTubeMusicVideoSuffix(t *testing.T) {
	request := Request{URL: "https://music.youtube.com/watch?v=track", Type: MediaVideo}
	info := Info{Title: "Tên Video", Artist: "Tên Ca Sĩ"}
	if got, want := outputFileStem(request, info), "Tên_Video_video"; got != want {
		t.Fatalf("outputFileStem() = %q, want %q", got, want)
	}
}

func TestOutputFileStemUsesInspectedWebpageURL(t *testing.T) {
	request := Request{URL: "https://youtu.be/track", Type: MediaAudio}
	info := Info{Title: "Tên Bài Hát", Artist: "Tên Ca Sĩ", WebpageURL: "https://music.youtube.com/watch?v=track"}
	if got, want := outputFileStem(request, info), "Tên Bài Hát - Tên Ca Sĩ"; got != want {
		t.Fatalf("outputFileStem() = %q, want %q", got, want)
	}
}

func TestOutputFileStemKeepsYouTubeTypeSuffix(t *testing.T) {
	request := Request{URL: "https://youtu.be/track", Type: MediaAudio}
	info := Info{Title: "Tên Bài Hát", Artist: "Tên Ca Sĩ"}
	if got, want := outputFileStem(request, info), "Tên_Bài_Hát_audio"; got != want {
		t.Fatalf("outputFileStem() = %q, want %q", got, want)
	}
}

func TestRawInfoArtistFallbackOrder(t *testing.T) {
	cases := []struct {
		name string
		raw  rawInfo
		want string
	}{
		{name: "artist first", raw: rawInfo{Artist: "Primary Artist", AlbumArtist: "Album Artist", Creator: "Creator"}, want: "Primary Artist"},
		{name: "artists list", raw: rawInfo{Artists: []string{"Artist One", "Artist Two"}, AlbumArtist: "Album Artist"}, want: "Artist One, Artist Two"},
		{name: "album artist", raw: rawInfo{AlbumArtist: "Album Artist", Creator: "Creator"}, want: "Album Artist"},
		{name: "creator", raw: rawInfo{Creator: "Creator"}, want: "Creator"},
		{name: "missing", raw: rawInfo{Uploader: "Channel"}, want: ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := firstNonEmpty(testCase.raw.Artist, strings.Join(testCase.raw.Artists, ", "), testCase.raw.AlbumArtist, testCase.raw.Creator)
			if got != testCase.want {
				t.Fatalf("artist fallback = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestSanitizeMusicPartPreservesReadableSeparator(t *testing.T) {
	if got, want := sanitizeMusicPart("  Bài / Hát  "), "Bài Hát"; got != want {
		t.Fatalf("sanitizeMusicPart() = %q, want %q", got, want)
	}
}
