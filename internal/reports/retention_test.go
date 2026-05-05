package reports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kigrepair/internal/app"
)

func TestValidateReportDeleteTargetSafety(t *testing.T) {
	root := t.TempDir()
	valid := makeReportDir(t, root, "2026-01-01_10-00-00", "summary.txt")
	outside := makeReportDir(t, t.TempDir(), "outside", "summary.txt")
	fileTarget := filepath.Join(root, "file.txt")
	if err := os.WriteFile(fileTarget, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := makeReportDir(t, filepath.Join(root, "parent"), "child", "summary.txt")
	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		target  string
		wantErr string
	}{
		{name: "accepts valid direct report directory", target: valid},
		{name: "rejects target outside reports root", target: outside, wantErr: "direct child"},
		{name: "rejects traversal", target: filepath.Join(root, "..", filepath.Base(outside)), wantErr: "direct child"},
		{name: "rejects reports root itself", target: root, wantErr: "reports root"},
		{name: "rejects parent", target: filepath.Dir(root), wantErr: "parent"},
		{name: "rejects direct file target", target: fileTarget, wantErr: "not a directory"},
		{name: "rejects nested child", target: nested, wantErr: "direct child"},
		{name: "rejects directory without known report files", target: empty, wantErr: "known report files"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReportDeleteTarget(root, tt.target, "")
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateReportDeleteTarget() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportDeleteTargetRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	real := makeReportDir(t, t.TempDir(), "real", "summary.txt")
	link := filepath.Join(root, "2026-01-01_10-00-00")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	err := ValidateReportDeleteTarget(root, link, "")
	if err == nil || !strings.Contains(err.Error(), "reparse point") {
		t.Fatalf("error = %v, want reparse point rejection", err)
	}
}

func TestBuildReportCleanupPlanRetentionPolicy(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	oldest := makeReportDir(t, root, "2026-03-01_10-00-00", "summary.txt")
	secondOld := makeReportDir(t, root, "2026-03-02_10-00-00", "operations.json")
	newestOld := makeReportDir(t, root, "2026-03-03_10-00-00", "repair-result.json")
	recent := makeReportDir(t, root, "2026-05-04_10-00-00", "summary.txt")
	nonReport := filepath.Join(root, "2026-02-01_10-00-00")
	if err := os.MkdirAll(nonReport, 0755); err != nil {
		t.Fatal(err)
	}
	active := makeReportDir(t, root, "2026-01-01_10-00-00", "summary.txt")

	plan, err := BuildReportCleanupPlan(root, ReportCleanupOptions{
		OlderThan:       "30d",
		KeepLast:        2,
		DryRun:          true,
		ActiveReportDir: active,
		Now:             now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("BuildReportCleanupPlan() error = %v", err)
	}
	assertPlanned(t, plan, oldest, secondOld)
	assertNotPlanned(t, plan, newestOld)
	assertNotPlanned(t, plan, recent)
	assertNotPlanned(t, plan, nonReport)
	assertNotPlanned(t, plan, active)
	if !skipContains(plan.Skipped, active, "active") {
		t.Fatalf("active report was not skipped: %#v", plan.Skipped)
	}
	if !skipContains(plan.Skipped, nonReport, "known report") {
		t.Fatalf("non-report directory was not skipped: %#v", plan.Skipped)
	}
}

func TestParseRetentionDuration(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
	}{
		{value: "30d", want: 30 * 24 * time.Hour},
		{value: "12h", want: 12 * time.Hour},
		{value: "720h", want: 720 * time.Hour},
	}
	for _, tt := range tests {
		got, err := ParseRetentionDuration(tt.value)
		if err != nil {
			t.Fatalf("ParseRetentionDuration(%q) error = %v", tt.value, err)
		}
		if got != tt.want {
			t.Fatalf("ParseRetentionDuration(%q) = %s, want %s", tt.value, got, tt.want)
		}
	}
}

func TestExecuteReportCleanupPlanDryRunDeletesNothing(t *testing.T) {
	root := t.TempDir()
	target := makeReportDir(t, root, "2026-01-01_10-00-00", "summary.txt")
	plan := ReportCleanupPlan{
		ReportsRoot: root,
		ReportDir:   makeReportDir(t, root, "2026-05-05_10-00-00", "summary.txt"),
		DryRun:      true,
		PlannedDeletes: []ReportDeleteTarget{{
			ID:        "1",
			Name:      filepath.Base(target),
			Path:      target,
			Validated: true,
		}},
	}
	result := ExecuteReportCleanupPlan(root, plan, false)
	if result.Status != "success" || result.ExitCode != 0 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("dry-run deleted target: %v", err)
	}
}

func TestExecuteReportCleanupPlanRequiresYesAndUsesOnlyPlan(t *testing.T) {
	root := t.TempDir()
	target := makeReportDir(t, root, "2026-01-01_10-00-00", "summary.txt")
	unplanned := makeReportDir(t, root, "2026-01-02_10-00-00", "summary.txt")
	active := makeReportDir(t, root, "2026-05-05_10-00-00", "summary.txt")
	plan := ReportCleanupPlan{
		ReportsRoot: root,
		ReportDir:   active,
		PlannedDeletes: []ReportDeleteTarget{{
			ID:        "1",
			Name:      filepath.Base(target),
			Path:      target,
			Validated: true,
		}},
	}
	missingYes := ExecuteReportCleanupPlan(root, plan, false)
	if missingYes.Status != "failed" || missingYes.ExitCode != 7 {
		t.Fatalf("missing yes result = %#v", missingYes)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target deleted without --yes: %v", err)
	}
	result := ExecuteReportCleanupPlan(root, plan, true)
	if result.Status != "success" || result.DeletedCount != 0 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target still exists or unexpected error: %v", err)
	}
	if _, err := os.Stat(unplanned); err != nil {
		t.Fatalf("unplanned target was touched: %v", err)
	}
}

func TestExecuteReportCleanupPlanPartialFailureWarning(t *testing.T) {
	root := t.TempDir()
	active := makeReportDir(t, root, "2026-05-05_10-00-00", "summary.txt")
	missing := filepath.Join(root, "2026-01-01_10-00-00")
	plan := ReportCleanupPlan{
		ReportsRoot: root,
		ReportDir:   active,
		PlannedDeletes: []ReportDeleteTarget{{
			ID:        "1",
			Name:      filepath.Base(missing),
			Path:      missing,
			Validated: true,
		}},
	}
	result := ExecuteReportCleanupPlan(root, plan, true)
	if result.Status != "warning" || result.ExitCode != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Deleted) != 1 || result.Deleted[0].Status != "failed" {
		t.Fatalf("delete outcomes = %#v", result.Deleted)
	}
}

func TestReportCleanupWorkflowWritesPlanResultAndSummary(t *testing.T) {
	root := t.TempDir()
	old := makeReportDir(t, root, "2026-01-01_10-00-00", "summary.txt")
	reportDir := makeReportDir(t, root, "2026-05-05_10-00-00", "summary.txt")
	ctx := testReportContext(t, root, reportDir)
	workflow := CleanupReportsWorkflow{OlderThan: "30d", KeepLast: 0, DryRun: true}
	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, name := range []string{"report-cleanup-plan.json", "report-cleanup-result.json", "operations.json", "summary.txt"} {
		if _, err := os.Stat(filepath.Join(reportDir, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(reportDir, "report-cleanup-plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan ReportCleanupPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	assertPlanned(t, plan, old)
	summary, err := os.ReadFile(filepath.Join(reportDir, "summary.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), "Policy: older than 30d, keep last 0") || !strings.Contains(string(summary), "Planned deletes: 1") {
		t.Fatalf("summary missing expected content:\n%s", string(summary))
	}
}

func TestListReportsJSONShapeAndSummaryRedaction(t *testing.T) {
	root := t.TempDir()
	dir := makeReportDir(t, root, "2026-01-01_10-00-00", "summary.txt")
	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte("Status: success\ninvite=secret-value\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ListReports(root, ReportListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"command":"reports list"`) || !strings.Contains(string(data), `"reports_root"`) {
		t.Fatalf("unexpected JSON shape: %s", string(data))
	}
	output := FormatReportList(result)
	if strings.Contains(output, "secret-value") {
		t.Fatalf("list output leaked invite: %s", output)
	}
}

func makeReportDir(t *testing.T, root string, name string, files ...string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("Status: success\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

type retentionTestLogger struct{}

func (retentionTestLogger) Info(string, ...any)  {}
func (retentionTestLogger) Warn(string, ...any)  {}
func (retentionTestLogger) Error(string, ...any) {}
func (retentionTestLogger) Close() error         { return nil }

func testReportContext(t *testing.T, root string, reportDir string) *app.AppContext {
	t.Helper()
	reporter, err := New(reportDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := app.NewContext()
	ctx.ReportRoot = root
	ctx.OutputDir = reportDir
	ctx.Reporter = reporter
	ctx.Logger = retentionTestLogger{}
	ctx.StartedAt = time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	return ctx
}

func assertPlanned(t *testing.T, plan ReportCleanupPlan, paths ...string) {
	t.Helper()
	for _, path := range paths {
		found := false
		for _, target := range plan.PlannedDeletes {
			if sameCleanPath(target.Path, path) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s was not planned for deletion: %#v", path, plan.PlannedDeletes)
		}
	}
}

func assertNotPlanned(t *testing.T, plan ReportCleanupPlan, path string) {
	t.Helper()
	for _, target := range plan.PlannedDeletes {
		if sameCleanPath(target.Path, path) {
			t.Fatalf("%s was unexpectedly planned for deletion", path)
		}
	}
}

func skipContains(skips []ReportSkip, path string, reason string) bool {
	for _, skip := range skips {
		if sameCleanPath(skip.Path, path) && strings.Contains(skip.Reason, reason) {
			return true
		}
	}
	return false
}
