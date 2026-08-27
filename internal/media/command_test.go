package media

import "testing"

func TestParseYTDCommand(t *testing.T) {
	request, matched, valid, err := ParseYTDCommand(".ytd https://youtu.be/dQw4w9WgXcQ?si=test type:video quality:137")
	if err != nil || !matched || !valid {
		t.Fatalf("expected valid video command, matched=%t valid=%t err=%v", matched, valid, err)
	}
	if request.Type != MediaVideo || request.Quality != "137" {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestParseYTDCommandDefaultsMusicToAudio(t *testing.T) {
	request, matched, valid, err := ParseYTDCommand(".ytd https://music.youtube.com/watch?v=test")
	if err != nil || !matched || !valid {
		t.Fatalf("expected valid music command, matched=%t valid=%t err=%v", matched, valid, err)
	}
	if request.Type != MediaAudio {
		t.Fatalf("expected music URL to default to audio, got %q", request.Type)
	}
}

func TestParseYTDCommandSupportsQualityList(t *testing.T) {
	request, matched, valid, err := ParseYTDCommand(".ytd https://www.youtube.com/watch?v=test type:video quality:list")
	if err != nil || !matched || !valid || request.Quality != "list" {
		t.Fatalf("expected quality list request, request=%#v matched=%t valid=%t err=%v", request, matched, valid, err)
	}
}

func TestParseYTDCommandWithCollectionPolicyAllowsCollectionsWhenDisabled(t *testing.T) {
	request, matched, valid, err := ParseYTDCommandWithCollectionPolicy(".ytd https://www.youtube.com/playlist?list=PL123 type:audio", false)
	if err != nil || !matched || !valid {
		t.Fatalf("expected collection request to be allowed when blocking is disabled, request=%#v matched=%t valid=%t err=%v", request, matched, valid, err)
	}
}

func TestParseYTDCommandRejectsPlaylistAndAlbumURLs(t *testing.T) {
	urls := []string{
		"https://www.youtube.com/playlist?list=PL123",
		"https://www.youtube.com/watch?v=video&list=PL123",
		"https://music.youtube.com/playlist?list=OLAK5uy_test",
		"https://music.youtube.com/browse/MPREb_test_album",
		"https://music.youtube.com/album/test",
	}
	for _, rawURL := range urls {
		if err := ValidateRequest(Request{URL: rawURL, Type: MediaAudio}); err == nil {
			t.Fatalf("ValidateRequest(%q) error = nil, want collection rejection", rawURL)
		}
	}
}

func TestValidateRequestAcceptsSingleYouTubeMediaURL(t *testing.T) {
	urls := []string{
		"https://www.youtube.com/watch?v=video",
		"https://youtu.be/video",
		"https://music.youtube.com/watch?v=song",
	}
	for _, rawURL := range urls {
		if err := ValidateRequest(Request{URL: rawURL, Type: MediaAudio}); err != nil {
			t.Fatalf("ValidateRequest(%q) error = %v, want nil", rawURL, err)
		}
	}
}

func TestParseYTDCommandRejectsOtherHosts(t *testing.T) {
	_, matched, valid, err := ParseYTDCommand(".ytd https://example.com/video type:video")
	if !matched || valid || err == nil {
		t.Fatalf("expected invalid host, matched=%t valid=%t err=%v", matched, valid, err)
	}
}

func TestParseYTDCommandRejectsQualityForThumbnail(t *testing.T) {
	_, matched, valid, err := ParseYTDCommand(".ytd https://youtu.be/dQw4w9WgXcQ type:thumbnail quality:137")
	if !matched || valid || err == nil {
		t.Fatalf("expected thumbnail quality error, matched=%t valid=%t err=%v", matched, valid, err)
	}
}
