package media

import (
	"strings"
	"testing"
)

func TestEstimateMediaSizeUsesSelectedFormat(t *testing.T) {
	request := Request{Type: MediaAudio, Quality: "251"}
	info := Info{Formats: []Format{
		{ID: "251", ACodec: "opus", FileSize: 12 * 1024 * 1024},
		{ID: "140", ACodec: "mp4a", FileSize: 8 * 1024 * 1024},
	}}
	if got, want := estimateMediaSize(request, info, 1024*1024*1024), int64(12*1024*1024); got != want {
		t.Fatalf("estimateMediaSize() = %d, want %d", got, want)
	}
}

func TestEstimateMediaSizeUsesMaxFileSizeFallback(t *testing.T) {
	request := Request{Type: MediaVideo, Quality: "best"}
	if got, want := estimateMediaSize(request, Info{}, 1024*1024*1024), int64(1024*1024*1024); got != want {
		t.Fatalf("estimateMediaSize() = %d, want %d", got, want)
	}
}

func TestRequiredDiskBytesIncludesSafetyMargin(t *testing.T) {
	request := Request{Type: MediaAudio, Quality: "251"}
	info := Info{Formats: []Format{{ID: "251", ACodec: "opus", FileSize: 10 * 1024 * 1024}}}
	if got, want := requiredDiskBytes(request, info, 1024*1024*1024, 2*1024*1024), int64(12*1024*1024); got != want {
		t.Fatalf("requiredDiskBytes() = %d, want %d", got, want)
	}
}

func TestFormatDiskBytes(t *testing.T) {
	if got, want := formatDiskBytes(2*1024*1024), "2.00 MiB"; got != want {
		t.Fatalf("formatDiskBytes() = %q, want %q", got, want)
	}
}

func TestEnsureDiskSpaceRejectsRequirementAboveAvailableSpace(t *testing.T) {
	path := t.TempDir()
	available, err := availableDiskBytes(path)
	if err != nil {
		t.Fatalf("availableDiskBytes() error = %v", err)
	}
	request := Request{Type: MediaAudio, Quality: "best"}
	if err := ensureDiskSpace(path, request, Info{}, 1, available); err == nil || !strings.Contains(err.Error(), "not enough disk space") {
		t.Fatalf("ensureDiskSpace() error = %v, want not-enough-space error", err)
	}
}
