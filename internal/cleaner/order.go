package cleaner

import "sort"

var cleanupActionOrder = map[CleanupActionType]int{
	CleanupActionStopService:       10,
	CleanupActionKillProcess:       20,
	CleanupActionMSIUninstall:      30,
	CleanupActionDeleteService:     40,
	CleanupActionDeletePath:        50,
	CleanupActionDeleteRegistryKey: 60,
}

func SortActionsForExecution(actions []CleanupAction) []CleanupAction {
	ordered := append([]CleanupAction(nil), actions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return actionOrder(ordered[i].Type) < actionOrder(ordered[j].Type)
	})
	return ordered
}

func actionOrder(actionType CleanupActionType) int {
	if order, ok := cleanupActionOrder[actionType]; ok {
		return order
	}
	return 1000
}
