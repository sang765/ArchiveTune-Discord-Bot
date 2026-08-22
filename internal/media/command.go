package media

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type MediaType string

const (
	MediaVideo     MediaType = "video"
	MediaAudio     MediaType = "audio"
	MediaThumbnail MediaType = "thumbnail"
)

type Request struct {
	URL     string
	Type    MediaType
	Quality string
}

var qualityPattern = regexp.MustCompile(`^[A-Za-z0-9+._-]+$`)

func ParseYTDCommand(content string) (request Request, matched, valid bool, err error) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) == 0 || !strings.EqualFold(fields[0], ".ytd") {
		return Request{}, false, false, nil
	}
	if len(fields) < 2 || len(fields) > 4 {
		return Request{}, true, false, fmt.Errorf("usage: `.ytd <YouTube URL> type:<video|audio|thumbnail> [quality:<format id>]`")
	}

	request.URL = fields[1]
	request.Type = defaultMediaType(request.URL)
	for _, field := range fields[2:] {
		key, value, ok := strings.Cut(field, ":")
		if !ok || strings.TrimSpace(value) == "" {
			return Request{}, true, false, fmt.Errorf("invalid option %q; use type:<video|audio|thumbnail> or quality:<format id>", field)
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "type":
			request.Type = MediaType(strings.ToLower(strings.TrimSpace(value)))
		case "quality":
			request.Quality = strings.TrimSpace(value)
		default:
			return Request{}, true, false, fmt.Errorf("unknown option %q", key)
		}
	}
	if err := ValidateRequest(request); err != nil {
		return Request{}, true, false, err
	}
	return request, true, true, nil
}

func ValidateRequest(request Request) error {
	if !IsYouTubeURL(request.URL) {
		return fmt.Errorf("URL must be a YouTube or YouTube Music link")
	}
	switch request.Type {
	case MediaVideo, MediaAudio, MediaThumbnail:
	default:
		return fmt.Errorf("type must be video, audio, or thumbnail")
	}
	if request.Quality != "" {
		if !qualityPattern.MatchString(request.Quality) {
			return fmt.Errorf("quality must be a format ID such as `251`, `140`, or `best`")
		}
	}
	if request.Type == MediaThumbnail && request.Quality != "" {
		return fmt.Errorf("thumbnail downloads do not use a quality value")
	}
	return nil
}

func IsYouTubeURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be", "www.youtu.be", "youtube-nocookie.com", "www.youtube-nocookie.com":
		return parsed.Path != ""
	default:
		return false
	}
}

func defaultMediaType(rawURL string) MediaType {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err == nil && strings.EqualFold(parsed.Hostname(), "music.youtube.com") {
		return MediaAudio
	}
	return MediaVideo
}
