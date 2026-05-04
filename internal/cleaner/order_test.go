package cleaner

import "testing"

func TestSortActionsForExecutionUsesRequiredOrder(t *testing.T) {
	actions := []CleanupAction{
		{Type: CleanupActionDeleteRegistryKey, Target: "registry"},
		{Type: CleanupActionDeletePath, Target: "path"},
		{Type: CleanupActionDeleteService, Target: "service-delete"},
		{Type: CleanupActionMSIUninstall, Target: "msi"},
		{Type: CleanupActionKillProcess, Target: "process"},
		{Type: CleanupActionStopService, Target: "service-stop"},
	}

	ordered := SortActionsForExecution(actions)
	want := []CleanupActionType{
		CleanupActionStopService,
		CleanupActionKillProcess,
		CleanupActionMSIUninstall,
		CleanupActionDeleteService,
		CleanupActionDeletePath,
		CleanupActionDeleteRegistryKey,
	}
	for i, actionType := range want {
		if ordered[i].Type != actionType {
			t.Fatalf("ordered[%d] = %s, want %s", i, ordered[i].Type, actionType)
		}
	}
}
