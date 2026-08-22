package media

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestUploadRetriesTransientServerErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("read upload body: %v", err)
		}
		attempt := requests.Add(1)
		if attempt < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("temporary failure"))
			return
		}
		_, _ = w.Write([]byte("https://temp.sh/test-file"))
	}))
	defer server.Close()

	fileName := writeUploadTestFile(t)
	downloader := NewDownloader("yt-dlp", "ffmpeg", t.TempDir())
	downloader.TempUploadURL = server.URL
	downloader.UploadTimeout = 10 * time.Second

	link, err := downloader.upload(context.Background(), fileName)
	if err != nil {
		t.Fatalf("upload() error = %v", err)
	}
	if link != "https://temp.sh/test-file" {
		t.Fatalf("upload() link = %q, want %q", link, "https://temp.sh/test-file")
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("upload() made %d requests, want 3", got)
	}
}

func TestUploadDoesNotRetryPermanentClientErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Errorf("read upload body: %v", err)
		}
		requests.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid request"))
	}))
	defer server.Close()

	downloader := NewDownloader("yt-dlp", "ffmpeg", t.TempDir())
	downloader.TempUploadURL = server.URL
	downloader.UploadTimeout = 10 * time.Second

	_, err := downloader.upload(context.Background(), writeUploadTestFile(t))
	if err == nil {
		t.Fatal("upload() error = nil, want HTTP 400 error")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("upload() made %d requests, want 1", got)
	}
}

func writeUploadTestFile(t *testing.T) string {
	t.Helper()
	fileName := t.TempDir() + "/sample.opus"
	if err := os.WriteFile(fileName, []byte("sample media data"), 0600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return fileName
}
