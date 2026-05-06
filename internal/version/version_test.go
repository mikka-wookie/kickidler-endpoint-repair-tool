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
		"signed":     info.SignedStatus,
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
	for _, field := range []string{`"tool"`, `"version"`, `"commit"`, `"build_date"`, `"built_by"`, `"go_version"`, `"os"`, `"arch"`, `"signed_status"`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("version JSON missing %s: %s", field, data)
		}
	}
}

func TestInjectedValuesAppearInVersionInfo(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate, oldBuiltBy := Version, Commit, BuildDate, BuiltBy
	t.Cleanup(func() {
		Version, Commit, BuildDate, BuiltBy = oldVersion, oldCommit, oldBuildDate, oldBuiltBy
	})
	Version = "1.2.3-test"
	Commit = "abc1234"
	BuildDate = "2026-05-06T12:00:00Z"
	BuiltBy = "builder@test"

	info := Get()
	if info.Version != Version || info.Commit != Commit || info.BuildDate != BuildDate || info.BuiltBy != BuiltBy {
		t.Fatalf("injected version values not reflected: %#v", info)
	}
}
