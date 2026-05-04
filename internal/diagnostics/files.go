package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type collectorError struct {
	collector string
	message   string
}

func (e *collectorError) Error() string {
	return fmt.Sprintf("%s: %s", e.collector, e.message)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func safeFilename(value string) string {
	replacer := strings.NewReplacer(
		`:\`, "_",
		`\`, "_",
		`/`, "_",
		`:`, "_",
		`*`, "_",
		`?`, "_",
		`"`, "_",
		`<`, "_",
		`>`, "_",
		`|`, "_",
		` `, "_",
	)
	name := replacer.Replace(value)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return strings.Trim(name, "_.")
}

func copyFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
