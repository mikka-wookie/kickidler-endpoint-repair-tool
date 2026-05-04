package reports

import (
	"os"
	"path/filepath"

	"kigrepair/internal/safety"
)

func WriteText(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(safety.RedactString(content)), 0644)
}
