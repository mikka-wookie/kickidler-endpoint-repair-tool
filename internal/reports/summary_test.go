package reports

import (
	"strings"
	"testing"
	"time"

	"kigrepair/internal/app"
)

func TestFormatSummaryIncludesRunMetadata(t *testing.T) {
	started := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	finished := started.Add(1500 * time.Millisecond)
	got := FormatSummary(SummaryData{
		Command:       "check",
		Started:       started,
		Finished:      finished,
		Mode:          "cli",
		ExitCode:      1,
		ReportDir:     `C:\ProgramData\kigrepair\Reports\2026-05-04_10-00-00`,
		InitialHealth: "broken",
		Warnings:      []string{"warning one"},
		Errors:        []string{"error one"},
		Actions:       []string{"Detection completed"},
	}, "Details\n")

	for _, want := range []string{
		"Command: check",
		"Started: 2026-05-04T10:00:00Z",
		"Duration: 1.5s",
		"Mode: cli",
		"Exit code: 1",
		"Initial health: broken",
		"- warning one",
		"- error one",
		"- Detection completed",
		"Details",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestFormatSummaryIncludesTimelineAndRedacts(t *testing.T) {
	started := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	got := FormatSummary(SummaryData{
		Command:   "repair",
		Started:   started,
		Finished:  started.Add(time.Second),
		ReportDir: `C:\ProgramData\kigrepair\Reports\2026-05-06_12-00-00`,
		Run: app.RunMetadata{
			RunID:         "kigrun-20260506-120000-a1b2c3",
			CorrelationID: "kigrun-20260506-120000-a1b2c3",
			WorkflowName:  "repair",
		},
		Operations: []app.OperationResult{
			{ID: "op-001-detect", Status: app.OperationStatusSuccess, Message: "Detection completed", DurationMS: 312},
			{ID: "op-002-preflight", Status: app.OperationStatusFailed, Message: "Installer validation failed: invite=REAL-SECRET-INVITE", DurationMS: 421, FailureCategory: "installer_validation"},
		},
	}, "Authorization: Bearer abc123\n")

	for _, want := range []string{
		"Run ID: kigrun-20260506-120000-a1b2c3",
		"Workflow timeline:",
		"1. [success] op-001-detect: Detection completed in 312 ms",
		"2. [failed] op-002-preflight: Installer validation failed: invite=<REDACTED> in 421 ms (failure: installer_validation)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "REAL-SECRET-INVITE") || strings.Contains(got, "abc123") {
		t.Fatalf("summary exposed secret:\n%s", got)
	}
}
