package app

import (
	"strings"
	"testing"
	"time"
)

func TestAddResultAssignsRunMetadataSequentialIDsAndProgress(t *testing.T) {
	ctx := NewContext()
	ctx.Run.RunID = "kigrun-20260506-120000-a1b2c3"
	ctx.Run.WorkflowName = "repair"
	ctx.Run.ReadOnly = true
	sink := &MemoryProgressSink{}
	ctx.Progress = sink

	started := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	ctx.AddResult(OperationResult{
		Step:       "repair.detect",
		Status:     OperationStatusSuccess,
		Message:    "Detection completed",
		StartedAt:  started,
		FinishedAt: started.Add(312 * time.Millisecond),
	})
	ctx.AddResult(OperationResult{
		Step:            "repair.preflight",
		Status:          OperationStatusFailed,
		Message:         "Preflight failed",
		FailureCategory: "installer_validation",
		StartedAt:       started,
		FinishedAt:      started.Add(time.Second),
	})

	if got := ctx.Results[0].ID; got != "op-001-detect" {
		t.Fatalf("first operation ID = %q", got)
	}
	if got := ctx.Results[1].ID; got != "op-002-preflight" {
		t.Fatalf("second operation ID = %q", got)
	}
	if ctx.Results[0].RunID != ctx.Run.RunID || ctx.Results[1].Workflow != "repair" {
		t.Fatalf("run metadata not copied to operations: %#v", ctx.Results)
	}
	if ctx.Results[0].DurationMS != 312 || ctx.Results[1].FailureCategory != "installer_validation" {
		t.Fatalf("operation timing/failure metadata not preserved: %#v", ctx.Results)
	}
	if !ctx.Results[0].ReadOnly {
		t.Fatalf("read_only was not inherited from run metadata")
	}
	if len(sink.Events) != 4 {
		t.Fatalf("progress events = %d, want 4", len(sink.Events))
	}
	if sink.Events[0].Stage != "operation_started" || sink.Events[1].Stage != "operation_finished" {
		t.Fatalf("unexpected progress stages: %#v", sink.Events[:2])
	}
}

func TestNewRunIDFormat(t *testing.T) {
	runID := NewRunID(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))
	if !strings.HasPrefix(runID, "kigrun-20260506-120000-") {
		t.Fatalf("RunID = %q", runID)
	}
}
