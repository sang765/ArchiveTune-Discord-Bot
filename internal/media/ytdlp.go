package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Format struct {
	ID       string
	Ext      string
	Note     string
	Width    int
	Height   int
	FPS      float64
	VCodec   string
	ACodec   string
	Bitrate  float64
	FileSize int64
}

type Info struct {
	ID         string
	Title      string
	Uploader   string
	Duration   float64
	WebpageURL string
	Thumbnail  string
	Formats    []Format
}

type Result struct {
	Info        Info
	FileName    string
	FileSize    int64
	DownloadURL string
}

type Downloader struct {
	YTDLPPath          string
	FFmpegPath         string
	TempUploadURL      string
	WorkDir            string
	InspectTimeout     time.Duration
	DownloadTimeout    time.Duration
	UploadTimeout      time.Duration
	MaxFileSize        int64
	MaxDurationSeconds int64
	semaphore          chan struct{}
	once               sync.Once
}

func NewDownloader(ytdlpPath, ffmpegPath, workDir string) *Downloader {
	return &Downloader{
		YTDLPPath:          ytdlpPath,
		FFmpegPath:         ffmpegPath,
		TempUploadURL:      "https://temp.sh/upload",
		WorkDir:            workDir,
		InspectTimeout:     60 * time.Second,
		DownloadTimeout:    20 * time.Minute,
		UploadTimeout:      20 * time.Minute,
		MaxFileSize:        1024 * 1024 * 1024,
		MaxDurationSeconds: 2 * 60 * 60,
		semaphore:          make(chan struct{}, 1),
	}
}

func (d *Downloader) Inspect(ctx context.Context, request Request) (Info, error) {
	if err := ValidateRequest(request); err != nil {
		return Info{}, err
	}
	if d.YTDLPPath == "" {
		return Info{}, fmt.Errorf("yt-dlp is not installed")
	}
	ctx, cancel := context.WithTimeout(ctx, d.InspectTimeout)
	defer cancel()
	args := []string{"--ignore-config", "--no-playlist", "--no-warnings", "--dump-single-json", "--skip-download", request.URL}
	output, err := d.run(ctx, args...)
	if err != nil {
		return Info{}, fmt.Errorf("inspect media: %w", err)
	}
	var raw rawInfo
	if err := json.Unmarshal(output, &raw); err != nil {
		return Info{}, fmt.Errorf("parse yt-dlp metadata: %w", err)
	}
	info := Info{
		ID:         raw.ID,
		Title:      raw.Title,
		Uploader:   raw.Uploader,
		Duration:   raw.Duration,
		WebpageURL: raw.WebpageURL,
		Thumbnail:  raw.Thumbnail,
		Formats:    make([]Format, 0, len(raw.Formats)),
	}
	for _, format := range raw.Formats {
		info.Formats = append(info.Formats, Format{
			ID:       format.FormatID,
			Ext:      format.Ext,
			Note:     format.FormatNote,
			Width:    format.Width,
			Height:   format.Height,
			FPS:      format.FPS,
			VCodec:   format.VCodec,
			ACodec:   format.ACodec,
			Bitrate:  format.TBR,
			FileSize: firstPositive(format.Filesize, format.FilesizeApprox),
		})
	}
	sort.SliceStable(info.Formats, func(i, j int) bool {
		if info.Formats[i].Height != info.Formats[j].Height {
			return info.Formats[i].Height > info.Formats[j].Height
		}
		return info.Formats[i].Bitrate > info.Formats[j].Bitrate
	})
	return info, nil
}

func (d *Downloader) DownloadAndUpload(ctx context.Context, request Request, info Info) (Result, error) {
	if err := ValidateRequest(request); err != nil {
		return Result{}, err
	}
	if info.Duration > 0 && d.MaxDurationSeconds > 0 && int64(info.Duration) > d.MaxDurationSeconds {
		return Result{}, fmt.Errorf("media duration exceeds the %d-minute limit", d.MaxDurationSeconds/60)
	}
	if d.YTDLPPath == "" {
		return Result{}, fmt.Errorf("yt-dlp is not installed")
	}
	select {
	case d.semaphore <- struct{}{}:
		defer func() { <-d.semaphore }()
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}

	tempDir, err := os.MkdirTemp(d.WorkDir, "ytd-")
	if err != nil {
		return Result{}, fmt.Errorf("create media workspace: %w", err)
	}
	defer os.RemoveAll(tempDir)

	ctx, cancel := context.WithTimeout(ctx, d.DownloadTimeout)
	defer cancel()
	outputTemplate := filepath.Join(tempDir, "media.%(ext)s")
	args := []string{"--ignore-config", "--no-playlist", "--no-warnings", "--no-overwrites", "--retries", "3", "--socket-timeout", "30", "--concurrent-fragments", "1", "--restrict-filenames", "--output", outputTemplate}
	if d.MaxFileSize > 0 {
		args = append(args, "--max-filesize", strconv.FormatInt(d.MaxFileSize/(1024*1024), 10)+"M")
	}
	if d.FFmpegPath != "" {
		args = append(args, "--ffmpeg-location", d.FFmpegPath)
	}
	switch request.Type {
	case MediaThumbnail:
		args = append(args, "--write-thumbnail", "--skip-download")
	case MediaAudio:
		args = append(args, "--format", audioFormat(request.Quality), "--extract-audio", "--audio-format", "best", "--embed-metadata", "--embed-thumbnail")
	default:
		args = append(args, "--format", videoFormat(request.Quality, info), "--embed-metadata", "--embed-thumbnail")
	}
	args = append(args, request.URL)
	if _, err := d.run(ctx, args...); err != nil {
		return Result{}, fmt.Errorf("download media: %w", err)
	}

	fileName, fileSize, err := findDownloadedFile(tempDir)
	if err != nil {
		return Result{}, err
	}
	if d.MaxFileSize > 0 && fileSize > d.MaxFileSize {
		return Result{}, fmt.Errorf("downloaded file is larger than the %d MB limit", d.MaxFileSize/(1024*1024))
	}
	link, err := d.upload(ctx, fileName)
	if err != nil {
		return Result{}, err
	}
	return Result{Info: info, FileName: filepath.Base(fileName), FileSize: fileSize, DownloadURL: link}, nil
}

func (d *Downloader) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.YTDLPPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("%s", detail)
	}
	return output, nil
}

func (d *Downloader) upload(ctx context.Context, fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", fmt.Errorf("open downloaded file: %w", err)
	}
	defer file.Close()

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	errCh := make(chan error, 1)
	go func() {
		defer writer.Close()
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(fileName))
		if err != nil {
			errCh <- err
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errCh <- err
			return
		}
		errCh <- multipartWriter.Close()
	}()

	ctx, cancel := context.WithTimeout(ctx, d.UploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.TempUploadURL, reader)
	if err != nil {
		return "", fmt.Errorf("create temp.sh request: %w", err)
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload to temp.sh: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err := <-errCh; err != nil {
		return "", fmt.Errorf("read downloaded file for upload: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("temp.sh returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	link := strings.TrimSpace(string(body))
	parsed, err := url.Parse(link)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "temp.sh" || parsed.Path == "" {
		return "", fmt.Errorf("temp.sh returned an invalid download URL")
	}
	return link, nil
}

func findDownloadedFile(dir string) (string, int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0, fmt.Errorf("read downloaded media: %w", err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".part") || strings.HasSuffix(entry.Name(), ".ytdl") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	if len(files) != 1 {
		return "", 0, fmt.Errorf("yt-dlp produced %d output files; expected exactly one", len(files))
	}
	stat, err := os.Stat(files[0])
	if err != nil {
		return "", 0, fmt.Errorf("stat downloaded media: %w", err)
	}
	return files[0], stat.Size(), nil
}

func videoFormat(quality string, info Info) string {
	if quality == "" || strings.EqualFold(quality, "best") {
		return "bestvideo*+bestaudio/best"
	}
	for _, format := range info.Formats {
		if format.ID == quality && format.VCodec != "" && format.VCodec != "none" && format.ACodec != "" && format.ACodec != "none" {
			return quality
		}
	}
	return quality + "+bestaudio/best"
}

func audioFormat(quality string) string {
	if quality == "" || strings.EqualFold(quality, "best") {
		return "bestaudio/best"
	}
	return quality
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

type rawInfo struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Uploader   string      `json:"uploader"`
	Duration   float64     `json:"duration"`
	WebpageURL string      `json:"webpage_url"`
	Thumbnail  string      `json:"thumbnail"`
	Formats    []rawFormat `json:"formats"`
}

type rawFormat struct {
	FormatID       string  `json:"format_id"`
	Ext            string  `json:"ext"`
	FormatNote     string  `json:"format_note"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	FPS            float64 `json:"fps"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	TBR            float64 `json:"tbr"`
	Filesize       int64   `json:"filesize"`
	FilesizeApprox int64   `json:"filesize_approx"`
}

func FormatLabel(format Format) string {
	parts := []string{format.ID, format.Ext}
	if format.Width > 0 && format.Height > 0 {
		parts = append(parts, strconv.Itoa(format.Width)+"x"+strconv.Itoa(format.Height))
	} else if format.ACodec != "none" && format.Bitrate > 0 {
		parts = append(parts, fmt.Sprintf("%.0fkbps", format.Bitrate))
	}
	if format.Note != "" {
		parts = append(parts, format.Note)
	}
	if format.VCodec != "none" && format.ACodec != "none" {
		parts = append(parts, "video+audio")
	} else if format.VCodec != "none" {
		parts = append(parts, "video-only")
	} else if format.ACodec != "none" {
		parts = append(parts, "audio-only")
	}
	return strings.Join(parts, " · ")
}

func FormatSummary(info Info, mediaType MediaType) string {
	if mediaType == MediaThumbnail {
		return "Thumbnail mode does not require a quality selection."
	}
	lines := make([]string, 0, 12)
	seen := make(map[string]struct{})
	for _, format := range info.Formats {
		if format.ID == "" || format.ID == "storyboard" {
			continue
		}
		isVideo := format.VCodec != "" && format.VCodec != "none"
		isAudio := format.ACodec != "" && format.ACodec != "none"
		if mediaType == MediaVideo && !isVideo {
			continue
		}
		if mediaType == MediaAudio && !isAudio {
			continue
		}
		if _, ok := seen[format.ID]; ok {
			continue
		}
		seen[format.ID] = struct{}{}
		lines = append(lines, "`"+FormatLabel(format)+"`")
		if len(lines) >= 15 {
			break
		}
	}
	if len(lines) == 0 {
		return "No selectable formats were returned by yt-dlp. Try `quality:best`."
	}
	return "Available quality formats:\n" + strings.Join(lines, "\n") + "\n\nRun the command again with `quality:<format id>`. Use `quality:best` for the highest available quality."
}
