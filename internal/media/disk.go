package media

import (
	"fmt"
	"math"
	"syscall"
)

const (
	defaultDiskSafetyMarginBytes int64 = 64 * 1024 * 1024
	defaultUnknownMediaSizeBytes int64 = 256 * 1024 * 1024
)

func availableDiskBytes(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

func requiredDiskBytes(request Request, info Info, maxFileSize, safetyMargin int64) int64 {
	estimated := estimateMediaSize(request, info, maxFileSize)
	if safetyMargin <= 0 {
		safetyMargin = defaultDiskSafetyMarginBytes
	}
	if estimated > math.MaxInt64-safetyMargin {
		return math.MaxInt64
	}
	return estimated + safetyMargin
}

func estimateMediaSize(request Request, info Info, maxFileSize int64) int64 {
	if request.Type == MediaThumbnail {
		return 8 * 1024 * 1024
	}

	var estimate int64
	for _, format := range info.Formats {
		if format.FileSize <= 0 {
			continue
		}
		isVideo := format.VCodec != "" && format.VCodec != "none"
		isAudio := format.ACodec != "" && format.ACodec != "none"
		if request.Type == MediaVideo && !isVideo {
			continue
		}
		if request.Type == MediaAudio && !isAudio {
			continue
		}
		if request.Quality != "" && request.Quality != "best" && format.ID != request.Quality {
			continue
		}
		if format.FileSize > estimate {
			estimate = format.FileSize
		}
	}
	if estimate > 0 {
		if maxFileSize > 0 && estimate > maxFileSize {
			return maxFileSize
		}
		return estimate
	}
	if maxFileSize > 0 {
		return maxFileSize
	}
	return defaultUnknownMediaSizeBytes
}

func ensureDiskSpace(path string, request Request, info Info, maxFileSize, safetyMargin int64) error {
	available, err := availableDiskBytes(path)
	if err != nil {
		return fmt.Errorf("check available disk space: %w", err)
	}
	required := requiredDiskBytes(request, info, maxFileSize, safetyMargin)
	if available < required {
		return fmt.Errorf("not enough disk space: %s available, at least %s required", formatDiskBytes(available), formatDiskBytes(required))
	}
	return nil
}

func formatDiskBytes(bytes int64) string {
	if bytes < 0 {
		return "unknown"
	}
	const unit = 1024.0
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	exponent := int(math.Log(float64(bytes)) / math.Log(unit))
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	if exponent >= len(units) {
		exponent = len(units) - 1
	}
	return fmt.Sprintf("%.2f %s", float64(bytes)/math.Pow(unit, float64(exponent)), units[exponent])
}
