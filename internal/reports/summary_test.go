package reports

import (
	"strings"
	"testing"
	"time"
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
