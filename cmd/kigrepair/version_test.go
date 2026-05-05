package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionCommandJSONIncludesBuildFields(t *testing.T) {
	opts := &globalOptions{jsonOutput: true}
	cmd := versionCommand(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version --json failed: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("version output is not JSON: %v\n%s", err, out.String())
	}
	for _, field := range []string{"tool", "version", "commit", "build_date", "built_by", "go_version", "os", "arch"} {
		if strings.TrimSpace(payload[field]) == "" {
			t.Fatalf("version JSON field %q is empty: %#v", field, payload)
		}
	}
}
