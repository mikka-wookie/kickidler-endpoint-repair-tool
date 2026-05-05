package cleaner

import (
	"fmt"

	"kigrepair/internal/app"
	"kigrepair/internal/rollback"
)

func RollbackFileTargets(plan CleanupPlan) []string {
	targets := make([]string, 0)
	for _, action := range plan.Actions {
		if action.Type == CleanupActionDeletePath {
			targets = append(targets, action.Target)
		}
	}
	return targets
}

func RollbackRegistryTargets(plan CleanupPlan) []string {
	targets := make([]string, 0)
	for _, action := range plan.Actions {
		if action.Type == CleanupActionDeleteRegistryKey {
			targets = append(targets, action.Target)
		}
	}
	return targets
}

func RollbackPlannedChanges(plan CleanupPlan, source string) []rollback.PlannedChange {
	changes := make([]rollback.PlannedChange, 0, len(plan.Actions))
	for idx, action := range plan.Actions {
		changeType := rollbackType(action.Type)
		warnings := []string{}
		if action.Error != "" {
			warnings = append(warnings, action.Error)
		}
		changes = append(changes, rollback.PlannedChange{
			ID:          fmt.Sprintf("%s-%03d", source, idx+1),
			Type:        changeType,
			Target:      action.Target,
			Destructive: changeType != rollback.TypeMSIInstall,
			Reason:      action.Reason,
			Source:      source,
			Warnings:    warnings,
		})
	}
	return changes
}

func RollbackExecutedChanges(operations []app.OperationResult, planned []rollback.PlannedChange) []rollback.ExecutedChange {
	changes := make([]rollback.ExecutedChange, 0)
	for _, operation := range operations {
		if !isExecutionResult(operation) {
			continue
		}
		changes = append(changes, rollback.ExecutedChangeFromOperation(operation, planned))
	}
	return changes
}

func rollbackType(actionType CleanupActionType) string {
	switch actionType {
	case CleanupActionMSIUninstall:
		return rollback.TypeMSIUninstall
	case CleanupActionStopService:
		return rollback.TypeServiceStop
	case CleanupActionDeleteService:
		return rollback.TypeServiceDelete
	case CleanupActionKillProcess:
		return rollback.TypeProcessTerminate
	case CleanupActionDeletePath:
		return rollback.TypeDirectoryDelete
	case CleanupActionDeleteRegistryKey:
		return rollback.TypeRegistryDelete
	default:
		return string(actionType)
	}
}
