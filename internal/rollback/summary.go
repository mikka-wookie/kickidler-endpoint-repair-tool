package rollback

import (
	"fmt"
	"path/filepath"
	"strings"
)

func Path(reportDir string) string {
	return filepath.Join(reportDir, "rollback-info.json")
}

func FormatSummarySection(info *RollbackInfo) string {
	var b strings.Builder
	b.WriteString("Rollback / Change Snapshot\n")
	b.WriteString("--------------------------\n")
	if info == nil {
		b.WriteString("Status: not created\n\n")
		return b.String()
	}
	b.WriteString("Snapshot file: rollback-info.json\n")
	if info.RestoreSupported {
		b.WriteString("Restore supported: yes\n")
	} else {
		b.WriteString("Restore supported: no\n")
	}
	b.WriteString(fmt.Sprintf("Services captured: %d\n", len(info.Before.Services)))
	b.WriteString(fmt.Sprintf("Processes captured: %d\n", len(info.Before.Processes)))
	b.WriteString(fmt.Sprintf("File targets captured: %d\n", len(info.Before.Files)+len(info.Before.Directories)))
	b.WriteString(fmt.Sprintf("Registry keys captured: %d\n", len(info.Before.RegistryKeys)))
	defenderCount := 0
	if info.Before.Defender != nil {
		defenderCount = len(info.Before.Defender.Exclusions)
	}
	b.WriteString(fmt.Sprintf("Defender exclusions captured: %d\n", defenderCount))
	b.WriteString(fmt.Sprintf("Planned changes: %d\n", len(info.PlannedChanges)))
	b.WriteString(fmt.Sprintf("Executed changes: %d\n\n", len(info.ExecutedChanges)))
	if len(info.Warnings) > 0 {
		b.WriteString("Warnings:\n")
		for _, warning := range info.Warnings {
			b.WriteString("- " + warning + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Note:\n")
	b.WriteString("Automatic rollback is not supported in this version. This snapshot is for audit and support escalation.\n\n")
	return b.String()
}
