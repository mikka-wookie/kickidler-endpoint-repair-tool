package rollback

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
)

func NewLedger(snapshot *RollbackInfo) *ChangeLedger {
	return &ChangeLedger{info: snapshot}
}

func RecordPlannedChange(ledger *ChangeLedger, change PlannedChange) {
	if ledger == nil || ledger.info == nil {
		return
	}
	if strings.TrimSpace(change.ID) == "" {
		change.ID = fmt.Sprintf("planned-%03d", len(ledger.info.PlannedChanges)+1)
	}
	ledger.info.PlannedChanges = append(ledger.info.PlannedChanges, change)
}

func RecordExecutedChange(ledger *ChangeLedger, change ExecutedChange) {
	if ledger == nil || ledger.info == nil {
		return
	}
	if strings.TrimSpace(change.FinishedAt) == "" {
		change.FinishedAt = nowString()
	}
	ledger.info.ExecutedChanges = append(ledger.info.ExecutedChanges, change)
}

func (l *ChangeLedger) Info() *RollbackInfo {
	if l == nil {
		return nil
	}
	return l.info
}

func ExecutedChangeFromOperation(operation app.OperationResult, planned []PlannedChange) ExecutedChange {
	changeType := ChangeTypeFromOperationStep(operation.Step)
	target := operation.Target
	plannedID := ""
	for _, plan := range planned {
		if plan.Type == changeType && strings.EqualFold(plan.Target, target) {
			plannedID = plan.ID
			break
		}
	}
	change := ExecutedChange{
		PlannedID:  plannedID,
		Type:       changeType,
		Target:     target,
		Status:     string(operation.Status),
		FinishedAt: operation.Timestamp.UTC().Format(time.RFC3339),
		Message:    operation.Message,
	}
	if operation.Status == app.OperationStatusFailed && operation.Error != "" {
		change.Errors = []string{operation.Error}
	}
	if operation.Status == app.OperationStatusWarning {
		if operation.Error != "" {
			change.Warnings = []string{operation.Error}
		} else if operation.Message != "" {
			change.Warnings = []string{operation.Message}
		}
	}
	return change
}

func ChangeTypeFromOperationStep(step string) string {
	switch step {
	case "stop_service":
		return TypeServiceStop
	case "delete_service":
		return TypeServiceDelete
	case "kill_process":
		return TypeProcessTerminate
	case "delete_path":
		return TypeDirectoryDelete
	case "delete_registry_key":
		return TypeRegistryDelete
	case "msi_uninstall":
		return TypeMSIUninstall
	case "msi_install":
		return TypeMSIInstall
	case "defender_exclusion_add", "defender_ensure":
		return TypeDefenderAddExclusion
	default:
		return step
	}
}

func SummaryFromInfo(path string, info *RollbackInfo) Summary {
	if info == nil {
		return Summary{SnapshotCreated: false, Path: path}
	}
	return Summary{
		SnapshotCreated:  true,
		Path:             path,
		RestoreSupported: info.RestoreSupported,
		PlannedChanges:   len(info.PlannedChanges),
		ExecutedChanges:  len(info.ExecutedChanges),
		Warnings:         append([]string{}, info.Warnings...),
	}
}
