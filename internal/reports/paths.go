package reports

import (
	"path/filepath"
	"time"

	"kigrepair/internal/config"
)

const TimestampLayout = "2006-01-02_15-04-05"

func TimestampedDir(root string, startedAt time.Time) string {
	if root == "" {
		root = config.DefaultReportRoot
	}
	return filepath.Join(root, startedAt.Format(TimestampLayout))
}

func OperationsPath(dir string) string {
	return filepath.Join(dir, "operations.json")
}

func LogPath(dir string) string {
	return filepath.Join(dir, config.DefaultLogName)
}
