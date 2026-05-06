package reports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"kigrepair/internal/app"
)

func TestWriteOperationsWritesOperationArray(t *testing.T) {
	reporter, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	results := []app.OperationResult{{
		ID:        "op-001-detect",
		RunID:     "kigrun-20260506-120000-a1b2c3",
		Step:      "check.detect",
		Target:    "grabber",
		Status:    app.OperationStatusSuccess,
		Message:   "Detection completed",
		Timestamp: time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC),
	}}
	if err := reporter.WriteOperations(results); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(reporter.Dir, "operations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded []app.OperationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0].Step != "check.detect" {
		t.Fatalf("decoded operations = %#v", decoded)
	}
	if decoded[0].RunID != "kigrun-20260506-120000-a1b2c3" || decoded[0].ID != "op-001-detect" {
		t.Fatalf("decoded operations missing observability IDs = %#v", decoded)
	}
}
