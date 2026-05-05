package testfixtures

import (
	"time"

	"kigrepair/internal/detector"
)

type FakeFile struct {
	Exists bool
	Size   int64
	Error  string
}

type FakeDirectory struct {
	Exists bool
	Error  string
}

func filesFromScenario(s Scenario) []detector.FileState {
	result := make([]detector.FileState, 0, len(s.Files)+len(s.Directories))
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	for path, file := range s.Files {
		result = append(result, detector.FileState{
			Path:         path,
			Exists:       file.Exists,
			Type:         "file",
			Size:         file.Size,
			ModifiedTime: now,
			Error:        file.Error,
		})
	}
	for path, dir := range s.Directories {
		result = append(result, detector.FileState{
			Path:         path,
			Exists:       dir.Exists,
			Type:         "folder",
			ModifiedTime: now,
			Error:        dir.Error,
		})
	}
	return result
}
