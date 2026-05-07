package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestWorkflowStatusForExit(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		err      error
		notReady bool
		want     string
	}{
		{name: "success", exitCode: ExitSuccess, want: "success"},
		{name: "warning", exitCode: ExitWarnings, want: "warning"},
		{name: "preflight not ready", exitCode: ExitInvalidInput, notReady: true, want: "not_ready"},
		{name: "missing installer", exitCode: ExitInvalidInput, notReady: true, want: "not_ready"},
		{name: "invalid installer", exitCode: ExitInvalidInput, notReady: true, want: "not_ready"},
		{name: "cancelled", exitCode: ExitInvalidInput, err: context.Canceled, want: "cancelled"},
		{name: "unexpected", exitCode: ExitUnexpectedError, err: errors.New("boom"), want: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkflowStatusForExit(tt.exitCode, tt.err, tt.notReady); got != tt.want {
				t.Fatalf("status = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUserMessageMappingUsesSupportReadableText(t *testing.T) {
	tests := []struct {
		text      string
		wantCode  string
		wantTitle string
	}{
		{text: "admin_rights=false", wantCode: "admin_rights", wantTitle: "Administrator rights required"},
		{text: "file_exists: file missing", wantCode: "installer_missing", wantTitle: "Installer file was not found"},
		{text: "preflight_not_ready", wantCode: "preflight_not_ready", wantTitle: "Repair is not ready"},
		{text: "defender_status_unavailable", wantCode: "defender_status_unavailable", wantTitle: "Defender status unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			msg := UserMessageFromText(tt.text, tt.text)
			if msg.Code != tt.wantCode || msg.Title != tt.wantTitle {
				t.Fatalf("message = %#v", msg)
			}
			if strings.Contains(msg.Title, tt.text) || strings.Contains(msg.Message, tt.text) {
				t.Fatalf("raw error used as primary message: %#v", msg)
			}
		})
	}
}

func TestOutcomeAggregatesProcessWarningsAndCapsTimeline(t *testing.T) {
	var warnings []string
	for i := 0; i < 91; i++ {
		warnings = append(warnings, "Skipped unsafe process match")
	}
	warnings = append(warnings,
		"normal Windows process path, not Grabber hidden WMI path",
		"hidden WMI process name but executable path is unavailable",
		"known Grabber process name but executable path is unavailable",
	)
	var ops []OperationResult
	for i := 0; i < 30; i++ {
		ops = append(ops, OperationResult{Step: "op", Status: OperationStatusSuccess, Message: "ok", Timestamp: time.Now()})
	}
	out := BuildWorkflowOutcome(OutcomeInput{
		Meta:     WorkflowResponseMeta{Workflow: "cleanup --dry-run", Status: "warning", ExitCode: ExitWarnings, ReportDir: `C:\Reports\run`, PrimaryResultFile: `C:\Reports\run\cleanup-plan.json`},
		Warnings: warnings,
		Timeline: ops,
	})
	assertMessageContains(t, out.Warnings, "Skipped unsafe process matches: 91.")
	assertMessageContains(t, out.Warnings, "Normal Windows process path mismatches skipped: 1.")
	assertMessageContains(t, out.Warnings, "Hidden WMI process names with unavailable paths: 1.")
	assertMessageContains(t, out.Warnings, "Known Grabber process names with unavailable paths skipped: 1.")
	if len(out.Timeline) != defaultOutcomeTimelineLimit+1 {
		t.Fatalf("timeline length = %d, want cap marker", len(out.Timeline))
	}
	if !strings.Contains(out.Timeline[len(out.Timeline)-1].Message, "operations.json") {
		t.Fatalf("missing details marker: %#v", out.Timeline[len(out.Timeline)-1])
	}
}

func TestRepairDryRunOutcomeContractAndRedaction(t *testing.T) {
	out := BuildWorkflowOutcome(OutcomeInput{
		Meta: WorkflowResponseMeta{
			RunID:             "run-1",
			Workflow:          "repair --dry-run",
			Status:            "failed",
			ExitCode:          ExitInvalidInput,
			ReportDir:         `C:\ProgramData\kigrepair\Reports\2026-05-07_20-18-24`,
			PrimaryResultFile: `C:\ProgramData\kigrepair\Reports\2026-05-07_20-18-24\repair-plan.json`,
		},
		Result: map[string]any{
			"repair_plan": map[string]any{
				"status":           "not_ready",
				"ready_for_repair": false,
				"detection_health": "unknown",
				"install_mode":     "unknown",
				"classification": map[string]any{
					"primary_issue": map[string]any{"code": "partial_msi_leftovers", "severity": "warning", "title": "MSI registry leftovers found"},
				},
				"recommendation": map[string]any{
					"primary_action": map[string]any{"code": "run_cleanup_dry_run", "title": "Run cleanup dry-run", "command": `.\kigrepair.exe repair --invite "REAL-SECRET-INVITE"`},
				},
				"preflight": map[string]any{
					"admin_rights": false,
					"checks": []any{
						map[string]any{"name": "admin_rights", "status": "fail", "required": true},
						map[string]any{"name": "installer_available", "status": "fail", "required": true, "error": "file_exists: file missing"},
					},
				},
				"installer": map[string]any{"installer_found": false, "installer_readable": false},
			},
		},
		Errors: []string{"admin_rights=false", "file_exists: file missing"},
	})
	if out.Status != "not_ready" || out.Health != "unknown" || out.InstallMode != "unknown" {
		t.Fatalf("unexpected outcome: %#v", out)
	}
	if out.PrimaryIssue == nil || out.PrimaryIssue.Code != "partial_msi_leftovers" {
		t.Fatalf("primary issue = %#v", out.PrimaryIssue)
	}
	if out.Recommendation == nil || out.Recommendation.Code != "run_cleanup_dry_run" {
		t.Fatalf("recommendation = %#v", out.Recommendation)
	}
	if !out.Admin.LimitedMode || out.Installer.Exists || out.Installer.Valid {
		t.Fatalf("admin/installer summaries = %#v %#v", out.Admin, out.Installer)
	}
	assertMessageCode(t, out.BlockingReasons, "admin_rights")
	assertMessageCode(t, out.BlockingReasons, "installer_missing")
	data, _ := json.Marshal(out)
	if strings.Contains(string(data), "REAL-SECRET-INVITE") {
		t.Fatalf("outcome leaked invite: %s", data)
	}
}

func assertMessageContains(t *testing.T, messages []UserMessage, want string) {
	t.Helper()
	for _, msg := range messages {
		if strings.Contains(msg.Message, want) {
			return
		}
	}
	t.Fatalf("message %q not found in %#v", want, messages)
}

func assertMessageCode(t *testing.T, messages []UserMessage, want string) {
	t.Helper()
	for _, msg := range messages {
		if msg.Code == want {
			return
		}
	}
	t.Fatalf("message code %q not found in %#v", want, messages)
}
