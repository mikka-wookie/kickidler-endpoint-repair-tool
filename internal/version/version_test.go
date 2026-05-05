package version

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetDefaultsAreNonEmpty(t *testing.T) {
	info := Get()
	if info.Tool != ToolName {
		t.Fatalf("tool = %q, want %q", info.Tool, ToolName)
	}
	values := map[string]string{
		"version":    info.Version,
		"commit":     info.Commit,
		"build_date": info.BuildDate,
		"built_by":   info.BuiltBy,
		"go_version": info.GoVersion,
		"os":         info.OS,
		"arch":       info.Arch,
	}
	for name, value := range values {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("%s is empty", name)
		}
	}
}

func TestInfoCanBeMarshaledToJSON(t *testing.T) {
	data, err := json.Marshal(Get())
	if err != nil {
		t.Fatalf("marshal version info: %v", err)
	}
	for _, field := range []string{`"tool"`, `"version"`, `"commit"`, `"build_date"`, `"built_by"`, `"go_version"`, `"os"`, `"arch"`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("version JSON missing %s: %s", field, data)
		}
	}
}
