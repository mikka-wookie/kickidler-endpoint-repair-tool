package repair

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/detector"
	"kigrepair/internal/reports"
)

func FormatDryRunConsole(plan RepairPlan) string {
	var b strings.Builder
	b.WriteString("Repair dry-run: " + plan.Status + "\n")
	b.WriteString("System changes made: no\n")
	b.WriteString("Report directory: " + plan.ReportDir + "\n\n")
	b.WriteString(fmt.Sprintf("Processes: %d trusted Grabber processes would be terminated, %d skipped unsafe name match.\n\n", plan.Cleanup.ProcessTerminateCount, plan.Cleanup.ProcessSkippedCount))
	if plan.ReadyForRepair {
		b.WriteString("Real repair is ready to run.\n\n")
	} else {
		b.WriteString("Planned real repair actions:\n")
		if plan.Cleanup.Required {
			b.WriteString("- Cleanup leftovers using cleanup-plan.json\n")
		}
		if plan.Defender.Required && plan.Defender.RequiredPath != "" {
			b.WriteString("- Ensure Defender exclusion for " + plan.Defender.RequiredPath + "\n")
		}
		if plan.Installer.WouldRunMsiexec && plan.Installer.InstallerPath != "" {
			b.WriteString("- Install MSI: " + plan.Installer.InstallerPath + "\n")
		} else if plan.Installer.Required {
			b.WriteString("- Install MSI: .\\grabberEM.x64.msi\n")
		}
		b.WriteString("- Verify final Grabber state\n\n")
	}
	if len(plan.Errors) > 0 {
		b.WriteString("Failed required checks:\n")
		for _, errText := range unique(plan.Errors) {
			b.WriteString("- " + errText + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Next command:\n")
	b.WriteString(plan.NextCommand + "\n")
	return b.String()
}

func FormatDryRunSummary(plan RepairPlan, initial *detector.DetectionReport) string {
	var b strings.Builder
	b.WriteString("Repair Dry-Run Plan\n")
	b.WriteString("-------------------\n")
	b.WriteString("Status: " + plan.Status + "\n")
	b.WriteString("Active profile: " + valueOrDash(plan.Policy.Profile) + "\n")
	b.WriteString("Ready for repair: " + yesNo(plan.ReadyForRepair) + "\n")
	b.WriteString("Dry-run only: yes\n")
	b.WriteString("System changes made: no\n")
	b.WriteString("Report directory: " + plan.ReportDir + "\n\n")
	b.WriteString("Initial detection:\n")
	b.WriteString("- Health: " + valueOrDash(plan.DetectionHealth) + "\n")
	b.WriteString("- Install mode: " + valueOrDash(plan.InstallMode) + "\n")
	b.WriteString("- Install root: " + valueOrDash(plan.InstallRoot) + "\n")
	if initial != nil {
		b.WriteString("- Primary service: " + valueOrDash(initial.PrimaryService) + "\n")
	}
	b.WriteString("\n")
	if plan.Preflight != nil {
		b.WriteString("Preflight:\n")
		for _, check := range plan.Preflight.Checks {
			b.WriteString(fmt.Sprintf("- %s: %s\n", check.Name, check.Status))
		}
		b.WriteString("\n")
	}
	b.WriteString("Cleanup:\n")
	b.WriteString("- Required: " + yesNo(plan.Cleanup.Required) + "\n")
	b.WriteString("- Destructive in real repair: " + yesNo(plan.Cleanup.Destructive) + "\n")
	b.WriteString(fmt.Sprintf("- Planned actions: %d\n", plan.Cleanup.ActionsCount))
	b.WriteString(fmt.Sprintf("- Process termination actions: %d trusted, %d skipped unsafe\n", plan.Cleanup.ProcessTerminateCount, plan.Cleanup.ProcessSkippedCount))
	b.WriteString("- Cleanup plan: cleanup-plan.json\n\n")
	b.WriteString("Defender:\n")
	b.WriteString("- Required path: " + valueOrDash(plan.Defender.RequiredPath) + "\n")
	b.WriteString("- Already covered: " + yesNo(plan.Defender.AlreadyCovered) + "\n")
	b.WriteString("- Would add exclusion in real repair: " + yesNo(plan.Defender.WouldAddExclusion) + "\n\n")
	b.WriteString("Installer:\n")
	b.WriteString("- Found: " + yesNo(plan.Installer.InstallerFound) + "\n")
	b.WriteString("- Path: " + valueOrDash(plan.Installer.InstallerPath) + "\n")
	if plan.Installer.Validation != nil {
		b.WriteString("- Validation: " + plan.Installer.Validation.Status + "\n")
		b.WriteString("- Architecture: " + valueOrDash(plan.Installer.Validation.Architecture) + "\n")
	}
	b.WriteString("- SHA-256: " + valueOrDash(plan.Installer.SHA256) + "\n")
	b.WriteString("- Would run msiexec: " + yesNo(plan.Installer.WouldRunMsiexec) + "\n")
	b.WriteString("- Invite present: " + yesNo(plan.Installer.HasInvite) + "\n\n")
	b.WriteString("Verification:\n")
	b.WriteString("- Would run after repair: yes\n")
	b.WriteString("- Checks: " + strings.Join(plan.Verification.Checks, ", ") + "\n\n")
	if len(plan.PolicyDecisions) > 0 || len(plan.BlockedByPolicy) > 0 {
		b.WriteString("Policy decisions:\n")
		for _, decision := range plan.PolicyDecisions {
			b.WriteString("- " + decision + "\n")
		}
		for _, blocker := range plan.BlockedByPolicy {
			b.WriteString("- blocked: " + blocker + "\n")
		}
		b.WriteString("\n")
	}
	if plan.Classification.PrimaryIssue != nil {
		b.WriteString("Classification:\n")
		b.WriteString("- Primary issue: " + plan.Classification.PrimaryIssue.Code + "\n\n")
	}
	if plan.Recommendation.PrimaryAction != nil {
		b.WriteString("Recommendation:\n")
		b.WriteString("- Primary action: " + plan.Recommendation.PrimaryAction.Code + "\n\n")
	}
	if len(plan.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range unique(plan.Warnings) {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}
	if len(plan.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, errText := range unique(plan.Errors) {
			b.WriteString("- " + errText + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Next command:\n")
	b.WriteString(plan.NextCommand + "\n")
	return reports.FormatSummary(reports.SummaryData{
		Command:       "repair --dry-run",
		Started:       time.Now(),
		Finished:      time.Now(),
		Mode:          "cli",
		ExitCode:      exitCodeForPlanStatus(plan.Status),
		ReportDir:     plan.ReportDir,
		InitialHealth: plan.DetectionHealth,
		InstallMode:   plan.InstallMode,
		InstallRoot:   plan.InstallRoot,
		Warnings:      plan.Warnings,
		Errors:        plan.Errors,
		Actions: []string{
			"Dry-run only: no system changes were made",
			fmt.Sprintf("Cleanup actions planned: %d", plan.Cleanup.ActionsCount),
			"Next command: " + plan.NextCommand,
		},
	}, b.String())
}
