package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/app"
)

func TestStructuredLoggerRedactsAndFiltersByLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repair.log")
	logger, err := NewWithOptions(path, Options{
		Quiet:    true,
		Level:    "info",
		RunID:    "kigrun-20260506-120000-a1b2c3",
		Workflow: "repair",
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("debug invite=REAL-SECRET-INVITE")
	logger.Info("command failed: invite=REAL-SECRET-INVITE Authorization: Bearer abc123 password=my-password access_token=abc refresh_token=def")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"REAL-SECRET-INVITE", "abc123", "my-password", "access_token=abc", "refresh_token=def"} {
		if strings.Contains(text, secret) {
			t.Fatalf("log exposed %q:\n%s", secret, text)
		}
	}
	if strings.Contains(text, "debug") {
		t.Fatalf("debug event was written at info level:\n%s", text)
	}
	var event Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &event); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, text)
	}
	if event.RunID != "kigrun-20260506-120000-a1b2c3" || event.Workflow != "repair" || event.Level != LevelInfo {
		t.Fatalf("unexpected event metadata: %#v", event)
	}
}

func TestLogOperationIncludesRunIDAndOperationID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repair.log")
	logger, err := NewWithOptions(path, Options{Quiet: true, Level: "debug"})
	if err != nil {
		t.Fatal(err)
	}
	LogOperation(logger, app.OperationResult{
		ID:        "op-001-detect",
		RunID:     "kigrun-20260506-120000-a1b2c3",
		Workflow:  "check",
		Step:      "check.detect",
		Target:    "grabber",
		Status:    app.OperationStatusSuccess,
		Message:   "Detection completed",
		Category:  app.OperationCategoryDetection,
		StartedAt: app.OperationResult{}.StartedAt,
	})
	_ = logger.Close()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"run_id":"kigrun-20260506-120000-a1b2c3"`) || !strings.Contains(text, `"operation_id":"op-001-detect"`) {
		t.Fatalf("operation log missing IDs:\n%s", text)
	}
}

func TestParseLevelRejectsInvalidLevel(t *testing.T) {
	if _, err := ParseLevel("verbose"); err == nil {
		t.Fatal("ParseLevel accepted invalid level")
	}
}
