package reports

import (
	"os"
	"path/filepath"

	"kigrepair/internal/safety"
)

func WriteJSON(path string, v any) error {
	data, err := marshalJSON(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, safety.RedactBytes(data), 0644)
}
