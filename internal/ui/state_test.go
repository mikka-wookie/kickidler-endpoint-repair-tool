package ui

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

const testInvite = "REAL-SECRET-INVITE"

type fakeService struct {
	delay      time.Duration
	repairSeen app.RepairRequest
	called     []Action
}

func (f *fakeService) Check(ctx context.Context, req app.CheckRequest) (*app.CheckResponse, error) {
	f.called = append(f.called, ActionCheck)
	return response("check", "success"), nil
}
func (f *fakeService) Verify(ctx context.Context, req app.VerifyRequest) (*app.VerifyResponse, error) {
	f.called = append(f.called, ActionVerify)
	return response("verify", "success"), nil
}
func (f *fakeService) Preflight(ctx context.Context, req app.PreflightRequest) (*app.PreflightResponse, error) {
	f.called = append(f.called, ActionPreflight)
	return responseWithTimeline("preflight", "invite="+req.InviteValue), nil
}
func (f *fakeService) RepairPlan(ctx context.Context, req app.RepairPlanRequest) (*app.RepairPlanResponse, error) {
	f.called = append(f.called, ActionRepairDryRun)
	return responseWithTimeline("repair --dry-run", "plan invite="+req.InviteValue), nil
}
func (f *fakeService) Repair(ctx context.Context, req app.RepairRequest) (*app.RepairResponse, error) {
	f.called = append(f.called, ActionRepair)
	f.repairSeen = req
	return responseWithTimeline("repair", "repair invite="+req.InviteValue), nil
}
func (f *fakeService) CleanupPlan(ctx context.Context, req app.CleanupPlanRequest) (*app.CleanupPlanResponse, error) {
	return response("cleanup-plan", "success"), nil
}
func (f *fakeService) Cleanup(ctx context.Context, req app.CleanupRequest) (*app.CleanupResponse, error) {
	return response("cleanup", "success"), nil
}
func (f *fakeService) CollectReport(ctx context.Context, req app.CollectReportRequest) (*app.CollectReportResponse, error) {
	return response("collect-report", "success"), nil
}
func (f *fakeService) ReportsList(ctx context.Context, req app.ReportsListRequest) (*app.ReportsListResponse, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return response("reports-list", "cancelled"), ctx.Err()
		}
	}
	return response("reports-list", "success"), nil
}
func (f *fakeService) ReportsCleanupPlan(ctx context.Context, req app.ReportsCleanupPlanRequest) (*app.ReportsCleanupPlanResponse, error) {
	return response("reports-cleanup-plan", "success"), nil
}
func (f *fakeService) ReportsCleanup(ctx context.Context, req app.ReportsCleanupRequest) (*app.ReportsCleanupResponse, error) {
	return response("reports-cleanup", "success"), nil
}
func (f *fakeService) ConfigShow(ctx context.Context, req app.ConfigShowRequest) (*app.ConfigShowResponse, error) {
	return response("config-show", "success"), nil
}
func (f *fakeService) ConfigValidate(ctx context.Context, req app.ConfigValidateRequest) (*app.ConfigValidateResponse, error) {
	return response("config-validate", "success"), nil
}
func (f *fakeService) GetWorkflowCatalog(context.Context) app.WorkflowCatalog {
	return app.DefaultWorkflowCatalog()
}

func TestResultRenderingDoesNotExposeInvite(t *testing.T) {
	svc := &fakeService{}
	controller := NewController(svc, nil)

	view, err := controller.Run(context.Background(), ActionRepairDryRun, Inputs{InstallerPath: "grabber.msi", InviteValue: testInvite})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	assertNoInvite(t, view)
}

func TestStartupViewDoesNotCallWorkflowService(t *testing.T) {
	svc := &fakeService{}
	controller := NewController(svc, nil)
	view := InitialResultView()
	if len(svc.called) != 0 {
		t.Fatalf("startup called workflows: %#v", svc.called)
	}
	if controller.IsRunning() {
		t.Fatal("startup should not mark controller running")
	}
	if view.Status != "Idle" || !strings.Contains(FormatTimeline(view.Timeline), "Start with Check") {
		t.Fatalf("unexpected startup view: %#v", view)
	}
}

func TestBuildResultViewExtractsSupportFields(t *testing.T) {
	resp := response("collect-report", "warning")
	resp.Result = map[string]any{
		"health":       "broken",
		"install_mode": "standard",
		"bundle_path":  filepath.Join(resp.Meta.ReportDir, "kigrepair-support-bundle.zip"),
		"classification": map[string]any{
			"primary_issue": map[string]any{"code": "service_binary_missing"},
		},
		"recommendation": map[string]any{
			"primary_action": map[string]any{"code": "collect_bundle", "message": "Collect support bundle and escalate."},
		},
	}
	view := BuildResultView(resp)
	if view.Classification != "broken" {
		t.Fatalf("health = %q", view.Classification)
	}
	if view.InstallMode != "standard" {
		t.Fatalf("install mode = %q", view.InstallMode)
	}
	if view.PrimaryIssueCode != "service_binary_missing" {
		t.Fatalf("primary issue = %q", view.PrimaryIssueCode)
	}
	if view.Recommendation != "Collect support bundle and escalate." {
		t.Fatalf("recommendation = %q", view.Recommendation)
	}
	if view.RecommendationCode != "collect_bundle" {
		t.Fatalf("recommendation code = %q", view.RecommendationCode)
	}
	if !strings.HasSuffix(view.SupportBundlePath, "kigrepair-support-bundle.zip") {
		t.Fatalf("support bundle path = %q", view.SupportBundlePath)
	}
}

func TestMapNotReadyRepairPlanToGUIState(t *testing.T) {
	resp := response("repair --dry-run", "warning")
	resp.Result = map[string]any{
		"repair_plan": map[string]any{
			"status":           "not_ready",
			"ready_for_repair": false,
			"detection_health": "unknown",
			"install_mode":     "unknown",
			"classification": map[string]any{
				"primary_issue": map[string]any{"code": "partial_msi_leftovers", "title": "Partial MSI leftovers"},
			},
			"recommendation": map[string]any{
				"primary_action": map[string]any{"code": "cleanup_dry_run", "message": "Run cleanup dry-run"},
			},
			"preflight": map[string]any{
				"checks": []any{
					map[string]any{"name": "admin_rights", "status": "failed", "required": true},
					map[string]any{"name": "installer_available", "status": "failed", "required": true},
				},
			},
		},
	}
	state := MapWorkflowResponseToGUIState(resp)
	if state.CurrentStatus != "Not ready" {
		t.Fatalf("status = %q", state.CurrentStatus)
	}
	if state.PrimaryIssueCode != "partial_msi_leftovers" {
		t.Fatalf("primary issue = %q", state.PrimaryIssueCode)
	}
	if state.PrimaryIssueTitle != "MSI registry leftovers found" {
		t.Fatalf("primary issue title = %q", state.PrimaryIssueTitle)
	}
	if state.RecommendationTitle != "Run cleanup dry-run" {
		t.Fatalf("recommendation = %q", state.RecommendationTitle)
	}
	if !contains(state.BlockingReasons, "Administrator rights required.") || !contains(state.BlockingReasons, "Installer file was not found.") {
		t.Fatalf("blocking reasons = %#v", state.BlockingReasons)
	}
	if state.Files.ReportDir == "" {
		t.Fatal("report dir was not mapped")
	}
}

func TestWarningAggregationCollapsesProcessFlood(t *testing.T) {
	var warnings []string
	for i := 0; i < 91; i++ {
		warnings = append(warnings, "Skipped unsafe process match")
	}
	items := AggregateTimelineWarnings(nil, warnings, "cleanup-plan.json")
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if !strings.Contains(items[0].Message, "91") || items[0].DetailsFile != "cleanup-plan.json" {
		t.Fatalf("aggregate item = %#v", items[0])
	}
}

func TestInviteNotStoredInGUIStateOrTimeline(t *testing.T) {
	resp := responseWithTimeline("repair --dry-run", "invite="+testInvite)
	resp.Warnings = []string{"warning invite=" + testInvite}
	resp.Errors = []string{"error invite=" + testInvite}
	resp.Result = map[string]any{"errors": []any{"result invite=" + testInvite}}
	view := BuildResultView(resp)
	assertNoInvite(t, view)
	state := MapWorkflowResponseToGUIState(resp)
	assertNoInvite(t, state)
}

func TestTimelineRenderingIncludesDurationAndFailureCategory(t *testing.T) {
	text := FormatTimeline([]TimelineItem{{
		Step:            "op-001-validate",
		Status:          string(app.OperationStatusFailed),
		Message:         "Installer validation failed",
		DurationMS:      421,
		FailureCategory: "installer",
	}})
	if !strings.Contains(text, "421 ms") || !strings.Contains(text, "installer") {
		t.Fatalf("timeline missing support details: %s", text)
	}
}

func TestRealRepairRequiresExactYES(t *testing.T) {
	svc := &fakeService{}
	controller := NewController(svc, func(context.Context, ConfirmationRequest) (bool, error) {
		return AcceptExactYES("yes"), nil
	})

	_, err := controller.Run(context.Background(), ActionRepair, Inputs{InstallerPath: "grabber.msi", InviteValue: testInvite})
	if err == nil {
		t.Fatal("expected confirmation failure")
	}
	if len(svc.called) != 0 {
		t.Fatalf("repair workflow was called without exact YES: %#v", svc.called)
	}
}

func TestRealRepairConfirmationRejectsCancelAndWhitespace(t *testing.T) {
	for _, input := range []string{"", " YES", "YES ", "Yes"} {
		if AcceptExactYES(input) {
			t.Fatalf("confirmation accepted %q", input)
		}
	}
}

func TestRealRepairAcceptedSetsYesAndRedactsInvite(t *testing.T) {
	svc := &fakeService{}
	controller := NewController(svc, func(context.Context, ConfirmationRequest) (bool, error) {
		return AcceptExactYES("YES"), nil
	})

	view, err := controller.Run(context.Background(), ActionRepair, Inputs{InstallerPath: "grabber.msi", InviteValue: testInvite})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !svc.repairSeen.Yes {
		t.Fatal("repair request did not set Yes after GUI confirmation")
	}
	assertNoInvite(t, view)
}

func TestOnlyOneWorkflowCanRunAtATimeAndCancelUsesContext(t *testing.T) {
	svc := &fakeService{delay: time.Second}
	controller := NewController(svc, nil)
	errCh := make(chan error, 1)
	go func() {
		_, err := controller.Run(context.Background(), ActionReportsList, Inputs{})
		errCh <- err
	}()

	deadline := time.Now().Add(time.Second)
	for !controller.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := controller.Run(context.Background(), ActionCheck, Inputs{}); err == nil {
		t.Fatal("expected concurrent workflow rejection")
	}
	controller.Cancel()
	err := <-errCh
	if !errors.Is(err, context.Canceled) && (err == nil || !strings.Contains(err.Error(), "canceled")) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestButtonStateWhileRunning(t *testing.T) {
	running := ComputeButtonState(true)
	if running.ActionsEnabled || !running.CancelEnabled {
		t.Fatalf("running button state = %#v", running)
	}
	idle := ComputeButtonState(false)
	if !idle.ActionsEnabled || idle.CancelEnabled {
		t.Fatalf("idle button state = %#v", idle)
	}
}

func TestOpenPathMissingFileReturnsSafeError(t *testing.T) {
	err := OpenPath(filepath.Join(t.TempDir(), "missing-summary.txt"))
	if err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestRequestStructsDoNotSerializeInvite(t *testing.T) {
	data, err := json.Marshal(app.RepairRequest{InstallerRequestFields: app.InstallerRequestFields{InstallerPath: "grabber.msi", InviteValue: testInvite}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), testInvite) {
		t.Fatalf("serialized request leaked invite: %s", data)
	}
}

func TestCatalogContainsRequiredGUIActions(t *testing.T) {
	actions := CatalogActionNames(app.DefaultWorkflowCatalog())
	for _, action := range []Action{ActionCheck, ActionVerify, ActionPreflight, ActionRepairDryRun, ActionRepair, ActionCollectReport, ActionReportsList, ActionReportsCleanupDry} {
		entry, ok := actions[action]
		if !ok {
			t.Fatalf("missing catalog action %s", action)
		}
		if action == ActionRepair {
			if !entry.Mutating || !entry.RequiresAdmin || !entry.RequiresInstaller || !entry.RequiresInvite {
				t.Fatalf("repair catalog safety metadata is incomplete: %#v", entry)
			}
			continue
		}
		if !entry.ReadOnly {
			t.Fatalf("%s should be read-only in catalog: %#v", action, entry)
		}
	}
}

func TestPackageBoundaryDoesNotImportCLIOrReverseUI(t *testing.T) {
	root := repoRoot(t)
	assertNoImportPrefix(t, filepath.Join(root, "internal", "ui"), "kigrepair/cmd/kigrepair")
	assertNoImportPrefix(t, filepath.Join(root, "internal", "ui"), "os/exec")
	assertNoImportPrefix(t, filepath.Join(root, "internal", "app"), "kigrepair/internal/ui")
}

func response(workflow string, status string) *app.WorkflowResponse {
	return &app.WorkflowResponse{
		Meta: app.WorkflowResponseMeta{
			RunID:             "run-1",
			Workflow:          workflow,
			Command:           workflow,
			Status:            status,
			ReportDir:         `C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22`,
			SummaryFile:       `C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\summary.txt`,
			OperationsFile:    `C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\operations.json`,
			PrimaryResultFile: `C:\ProgramData\kigrepair\Reports\2026-05-03_14-30-22\result.json`,
		},
	}
}

func responseWithTimeline(workflow string, message string) *app.WorkflowResponse {
	resp := response(workflow, "success")
	resp.Timeline = []app.OperationResult{{
		Step:      "test.step",
		Target:    "target",
		Status:    app.OperationStatusSuccess,
		Message:   message,
		Timestamp: time.Now(),
	}}
	return resp
}

func assertNoInvite(t *testing.T, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), testInvite) {
		t.Fatalf("invite leaked: %s", data)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertNoImportPrefix(t *testing.T, dir string, forbidden string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), `"`+forbidden) {
			t.Fatalf("%s imports forbidden package prefix %s", filepath.Join(dir, entry.Name()), forbidden)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
