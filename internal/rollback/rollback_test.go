package rollback

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

func TestCaptureSnapshotFromDetection(t *testing.T) {
	file := filepath.Join(t.TempDir(), "grabber.exe")
	if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	report := detector.DetectionReport{
		Health:      detector.GrabberHealthBroken,
		InstallMode: detector.InstallModeStandard,
		InstallRoot: `C:\Program Files\TeleLinkSoft`,
		IsAdmin:     true,
		Services: []detector.ServiceState{{
			Name:                     "ngs",
			Exists:                   true,
			Status:                   "running",
			StartType:                "auto",
			RawImagePath:             `"C:\Program Files\TeleLinkSoft\bin\grabber2.exe"`,
			NormalizedExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
			TrustLevel:               "trusted",
		}},
		Processes: []detector.ProcessState{{
			PID:                      12,
			Name:                     "grabber2.exe",
			NormalizedExecutablePath: `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
			TrustLevel:               detector.ProcessTrustNameAndPathMatch,
		}},
		Defender: detector.DefenderState{
			Available:                true,
			ExclusionPaths:           []string{`C:\Program Files\TeleLinkSoft`},
			NormalizedExclusionPaths: []string{`C:\Program Files\TeleLinkSoft`},
			RequiredPaths:            []string{`C:\Program Files\TeleLinkSoft\bin`},
		},
	}

	info, err := CaptureSnapshot(context.Background(), SnapshotInput{
		ReportDir:     t.TempDir(),
		Workflow:      WorkflowCleanup,
		Detection:     &report,
		FileTargets:   []string{file, filepath.Join(t.TempDir(), "missing.exe")},
		RequiredPaths: []string{`C:\Program Files\TeleLinkSoft\bin`},
		HasInvite:     true,
		IsAdmin:       true,
	})
	if err != nil {
		t.Fatalf("CaptureSnapshot() error = %v", err)
	}
	if info.Before.DetectionHealth != string(detector.GrabberHealthBroken) {
		t.Fatalf("health = %q", info.Before.DetectionHealth)
	}
	if len(info.Before.Services) != 1 || info.Before.Services[0].Name != "ngs" {
		t.Fatalf("services = %#v", info.Before.Services)
	}
	if len(info.Before.Processes) != 1 || info.Before.Processes[0].PID != 12 {
		t.Fatalf("processes = %#v", info.Before.Processes)
	}
	if len(info.Before.Files) != 2 {
		t.Fatalf("files = %#v", info.Before.Files)
	}
	if !info.Before.Files[0].Exists || info.Before.Files[0].SizeBytes != int64(len("content")) || info.Before.Files[0].SHA256 == "" {
		t.Fatalf("small file snapshot = %#v", info.Before.Files[0])
	}
	if info.Before.Files[1].Exists {
		t.Fatalf("missing file marked exists: %#v", info.Before.Files[1])
	}
	if info.Before.Defender == nil || !info.Before.Defender.Covered {
		t.Fatalf("defender snapshot = %#v", info.Before.Defender)
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "raw-invite") {
		t.Fatalf("raw invite leaked in rollback JSON")
	}
	if !info.HasInvite {
		t.Fatalf("HasInvite was not stored")
	}
}

func TestCaptureSnapshotSkipsLargeFileHash(t *testing.T) {
	file := filepath.Join(t.TempDir(), "large.bin")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxHashBytes + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	_ = f.Close()

	info, err := CaptureSnapshot(context.Background(), SnapshotInput{ReportDir: t.TempDir(), Workflow: WorkflowCleanup, FileTargets: []string{file}})
	if err != nil {
		t.Fatalf("CaptureSnapshot() error = %v", err)
	}
	if len(info.Before.Files) != 1 {
		t.Fatalf("files = %#v", info.Before.Files)
	}
	if info.Before.Files[0].SHA256 != "" || !strings.Contains(info.Before.Files[0].Error, "hash skipped") {
		t.Fatalf("large file snapshot = %#v", info.Before.Files[0])
	}
	if len(info.Warnings) == 0 {
		t.Fatalf("expected warning for skipped hash")
	}
}

func TestWriteFileCreatesRollbackInfoBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	info, err := CaptureSnapshot(context.Background(), SnapshotInput{
		ReportDir: dir,
		Workflow:  WorkflowCleanup,
		PlannedChanges: []PlannedChange{{
			ID:          "cleanup-001",
			Type:        TypeDirectoryDelete,
			Target:      `C:\Program Files\TeleLinkSoft`,
			Destructive: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(dir, info); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatalf("rollback-info.json was not created: %v", err)
	}
	if !strings.Contains(string(data), `"planned_changes"`) {
		t.Fatalf("planned changes missing from rollback-info.json")
	}
}

func TestLedgerRecordsExecutedChanges(t *testing.T) {
	info := &RollbackInfo{PlannedChanges: []PlannedChange{{ID: "cleanup-001", Type: TypeServiceDelete, Target: "ngs"}}}
	ledger := NewLedger(info)
	operation := app.OperationResult{
		Step:      "delete_service",
		Target:    "ngs",
		Status:    app.OperationStatusFailed,
		Message:   "delete failed",
		Error:     "access denied",
		Timestamp: time.Now(),
	}

	RecordExecutedChange(ledger, ExecutedChangeFromOperation(operation, info.PlannedChanges))

	if len(info.ExecutedChanges) != 1 {
		t.Fatalf("executed changes = %#v", info.ExecutedChanges)
	}
	got := info.ExecutedChanges[0]
	if got.PlannedID != "cleanup-001" || got.Type != TypeServiceDelete || got.Status != string(app.OperationStatusFailed) {
		t.Fatalf("executed change = %#v", got)
	}
	if len(got.Errors) != 1 || got.Errors[0] != "access denied" {
		t.Fatalf("errors = %#v", got.Errors)
	}
}
