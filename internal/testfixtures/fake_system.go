package testfixtures

import (
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/detector"
)

type MutationRecorder struct {
	Calls []MutationCall
}

type MutationCall struct {
	Type      string
	Target    string
	Timestamp time.Time
}

func (r *MutationRecorder) Record(callType string, target string) {
	r.Calls = append(r.Calls, MutationCall{Type: callType, Target: target, Timestamp: time.Now()})
}

func (r *MutationRecorder) ExecutePlan(plan cleaner.CleanupPlan) []app.OperationResult {
	results := make([]app.OperationResult, 0, len(plan.Actions))
	for _, action := range cleaner.SortActionsForExecution(plan.Actions) {
		r.Record(string(action.Type), action.Target)
		results = append(results, app.OperationResult{
			Step:      string(action.Type),
			Target:    action.Target,
			Status:    app.OperationStatusSuccess,
			Message:   "fake mutation recorded",
			Timestamp: time.Now(),
		})
	}
	return results
}

func (r *MutationRecorder) Types() []string {
	result := make([]string, 0, len(r.Calls))
	for _, call := range r.Calls {
		result = append(result, call.Type)
	}
	return result
}

func EnrichedReport(s Scenario) detector.DetectionReport {
	return s.DetectionReport()
}

func pathsEqual(left string, right string) bool {
	return strings.EqualFold(detector.NormalizeWindowsPath(left), detector.NormalizeWindowsPath(right))
}
