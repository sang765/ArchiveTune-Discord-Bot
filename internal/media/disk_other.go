//go:build !linux

package media

import "fmt"

func availableDiskBytes(path string) (int64, error) {
	return 0, fmt.Errorf("disk space check not supported on this platform")
}
