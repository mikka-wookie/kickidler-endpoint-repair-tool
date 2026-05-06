package workflowservice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/app"
)

type fakeWorkflow struct {
	name string
}

func (w fakeWorkflow) Name() string { return w.name }

func (w fakeWorkflow) Run(ctx *app.AppContext) error {
	ctx.JSONValue = map[string]any{"ok": true}
	ctx.AddResult(app.OperationResult{
		Step:      "fake.operation",
		Target:    "fake",
		Status:    app.OperationStatusSuccess,
		Message:   "completed invite=REAL-SECRET-INVITE",
		Timestamp: time.Now(),
	})
	ctx.ExitCode = app.ExitSuccess
	return ctx.Reporter.WriteText("summary", "fake summary")
}

func TestServiceConstructionAndCatalog(t *testing.T) {
	svc := NewDefault()
	catalog := svc.GetWorkflowCatalog(context.Background())
	if len(catalog.Workflows) == 0 {
		t.Fatal("catalog is empty")
	}
	requireEntry(t, catalog, "check", true, false, false, false, false)
	requireEntry(t, catalog, "repair", false, true, true, true, true)
	requireEntry(t, catalog, "cleanup", false, true, true, false, false)
}

func TestExecuteWorkflowProgressAndRedaction(t *testing.T) {
	dir := t.TempDir()
	sink := &app.MemoryProgressSink{}
	svc := New(app.WorkflowOptions{ProgressSink: sink})
	resp, err := svc.ExecuteWorkflow(context.Background(), app.CommonRequest{OutputDir: dir}, fakeWorkflow{name: "fake"}, "fake-result.json")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Meta.RunID == "" || resp.Meta.ReportDir == "" {
		t.Fatalf("missing response metadata: %#v", resp.Meta)
	}
	if len(sink.Events) < 4 {
		t.Fatalf("progress events = %d, want workflow and operation events", len(sink.Events))
	}
	encoded, _ := json.Marshal(resp)
	if strings.Contains(string(encoded), "REAL-SECRET-INVITE") {
		t.Fatalf("response leaked invite: %s", encoded)
	}
	for _, event := range sink.Events {
		data, _ := json.Marshal(event)
		if strings.Contains(string(data), "REAL-SECRET-INVITE") {
			t.Fatalf("progress leaked invite: %s", data)
		}
	}
	if resp.Meta.OperationsFile != filepath.Join(resp.Meta.ReportDir, "operations.json") {
		t.Fatalf("operations path = %q", resp.Meta.OperationsFile)
	}
}

func TestExecuteWorkflowRedactsSensitiveValuesFromResponse(t *testing.T) {
	dir := t.TempDir()
	svc := NewDefault()
	resp, err := svc.ExecuteWorkflowWithSensitive(context.Background(), app.CommonRequest{OutputDir: dir}, fakeWorkflow{name: "fake"}, "fake-result.json", []string{"REAL-SECRET-INVITE"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(resp)
	if strings.Contains(string(encoded), "REAL-SECRET-INVITE") {
		t.Fatalf("response leaked sensitive value: %s", encoded)
	}
	if !strings.Contains(string(encoded), "REDACTED") {
		t.Fatalf("response did not show redaction marker: %s", encoded)
	}
}

func TestExecuteWorkflowWritesReportsOnWorkflowFailure(t *testing.T) {
	dir := t.TempDir()
	svc := NewDefault()
	resp, err := svc.ExecuteWorkflow(context.Background(), app.CommonRequest{OutputDir: dir}, failingWorkflow{name: "fake-fail"}, "fake-result.json")
	if err == nil {
		t.Fatal("expected workflow error")
	}
	if resp.Meta.Status != string(app.WorkflowStatusFailed) {
		t.Fatalf("status = %q, want failed", resp.Meta.Status)
	}
	for _, path := range []string{resp.Meta.OperationsFile, resp.Meta.SummaryFile} {
		if path == "" {
			t.Fatalf("missing report path in response: %#v", resp.Meta)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("expected report file %s: %v", path, statErr)
		}
	}
}

func TestCancelledContextProducesCancelledResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := NewDefault()
	resp, err := svc.ExecuteWorkflow(ctx, app.CommonRequest{OutputDir: t.TempDir()}, fakeWorkflow{name: "fake"}, "fake-result.json")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Meta.Status != string(app.WorkflowStatusCancelled) {
		t.Fatalf("status = %q, want cancelled", resp.Meta.Status)
	}
	if resp.Meta.ExitCode != app.ExitInvalidInput {
		t.Fatalf("exit code = %d, want %d", resp.Meta.ExitCode, app.ExitInvalidInput)
	}
	if resp.Meta.ReportDir == "" || resp.Meta.OperationsFile == "" {
		t.Fatalf("cancelled response missing report references: %#v", resp.Meta)
	}
}

func requireEntry(t *testing.T, catalog app.WorkflowCatalog, name string, readOnly bool, mutating bool, admin bool, installer bool, invite bool) {
	t.Helper()
	for _, entry := range catalog.Workflows {
		if entry.Name != name {
			continue
		}
		if entry.ReadOnly != readOnly || entry.Mutating != mutating || entry.RequiresAdmin != admin || entry.RequiresInstaller != installer || entry.RequiresInvite != invite {
			t.Fatalf("catalog entry %s flags mismatch: %#v", name, entry)
		}
		return
	}
	t.Fatalf("catalog entry %q not found", name)
}

type failingWorkflow struct {
	name string
}

func (w failingWorkflow) Name() string { return w.name }

func (w failingWorkflow) Run(ctx *app.AppContext) error {
	ctx.AddResult(app.OperationResult{
		Step:      "fake.failed",
		Target:    "fake",
		Status:    app.OperationStatusFailed,
		Message:   "failed",
		Timestamp: time.Now(),
	})
	ctx.ExitCode = app.ExitUnexpectedError
	return errors.New("workflow failed")
}
