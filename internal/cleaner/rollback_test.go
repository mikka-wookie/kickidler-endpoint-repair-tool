package cleaner

import (
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/rollback"
)

func TestRollbackPlannedChangesFromCleanupPlan(t *testing.T) {
	plan := CleanupPlan{Actions: []CleanupAction{
		{Type: CleanupActionStopService, Target: "ngs", Safe: true, Reason: "service"},
		{Type: CleanupActionDeleteService, Target: "ngs", Safe: true, Reason: "service"},
		{Type: CleanupActionKillProcess, Target: "grabber.exe PID 10", Safe: true, Reason: "process"},
		{Type: CleanupActionDeletePath, Target: `C:\Program Files\TeleLinkSoft`, Safe: true, Reason: "path"},
		{Type: CleanupActionDeleteRegistryKey, Target: `HKLM\SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`, Safe: true, Reason: "registry"},
		{Type: CleanupActionMSIUninstall, Target: "{PRODUCT}", Safe: true, Reason: "msi"},
	}}

	changes := RollbackPlannedChanges(plan, rollback.WorkflowCleanup)
	wantTypes := []string{
		rollback.TypeServiceStop,
		rollback.TypeServiceDelete,
		rollback.TypeProcessTerminate,
		rollback.TypeDirectoryDelete,
		rollback.TypeRegistryDelete,
		rollback.TypeMSIUninstall,
	}
	if len(changes) != len(wantTypes) {
		t.Fatalf("changes = %#v", changes)
	}
	for i, want := range wantTypes {
		if changes[i].Type != want {
			t.Fatalf("change[%d].Type = %q, want %q", i, changes[i].Type, want)
		}
		if changes[i].ID == "" {
			t.Fatalf("change[%d] missing ID", i)
		}
	}
}

func TestRollbackExecutedChangesReferencesPlannedID(t *testing.T) {
	planned := []rollback.PlannedChange{{ID: "cleanup-001", Type: rollback.TypeProcessTerminate, Target: "grabber.exe PID 10"}}
	ops := []app.OperationResult{{
		Step:    string(CleanupActionKillProcess),
		Target:  "grabber.exe PID 10",
		Status:  app.OperationStatusSkipped,
		Message: "already exited",
	}}

	changes := RollbackExecutedChanges(ops, planned)
	if len(changes) != 1 {
		t.Fatalf("changes = %#v", changes)
	}
	if changes[0].PlannedID != "cleanup-001" || changes[0].Status != string(app.OperationStatusSkipped) {
		t.Fatalf("executed change = %#v", changes[0])
	}
}
