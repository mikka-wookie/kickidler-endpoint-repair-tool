package rollback

import (
	"bytes"
	"encoding/json"
	"os"
)

func writeJSON(path string, v any) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0644)
}
