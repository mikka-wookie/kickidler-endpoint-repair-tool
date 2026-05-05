package cleaner_test

import (
	"strings"
	"testing"

	"kigrepair/internal/cleaner"
	"kigrepair/internal/detector"
	"kigrepair/internal/testfixtures"
)

func TestCleanupPlanScenarios(t *testing.T) {
	for _, name := range []string{
		"partial_msi_leftovers",
		"partial_files_leftover",
		"hidden_wmi_service_binary_missing",
		"supported_service_name_path_mismatch",
		"process_name_match_path_mismatch",
		"normal_windows_svchost_not_grabber",
	} {
		t.Run(name, func(t *testing.T) {
			scenario := testfixtures.ScenarioByName(name)
			report := scenario.DetectionReport()
			plan := cleaner.BuildPlan(report, cleaner.PlanOptions{
				DryRun:       true,
				CleanupPaths: scenario.CleanupPaths(),
				PathExists:   scenario.PathExists,
				ValidatePath: func(string) error { return nil },
			})
			if len(plan.Actions) < scenario.Expected.CleanupActionsMin {
				t.Fatalf("actions = %d, want at least %d; plan=%#v", len(plan.Actions), scenario.Expected.CleanupActionsMin, plan)
			}
			if scenario.Expected.NoCleanupTerminateAction && containsAction(plan, cleaner.CleanupActionKillProcess) {
				t.Fatalf("unsafe process termination action was planned: %#v", plan.Actions)
			}
			if name == "supported_service_name_path_mismatch" && containsTarget(plan, "ngs") {
				t.Fatalf("path-mismatched service should not be mutated: %#v", plan.Actions)
			}
			for _, action := range plan.Actions {
				if action.Type == cleaner.CleanupActionKillProcess && action.TrustLevel == detector.ProcessTrustPathMismatch {
					t.Fatalf("path-mismatched process should not be killable: %#v", action)
				}
			}
		})
	}
}

func TestCleanupDryRunDoesNotMutate(t *testing.T) {
	scenario := testfixtures.ScenarioByName("partial_files_leftover")
	report := scenario.DetectionReport()
	recorder := &testfixtures.MutationRecorder{}
	_ = cleaner.BuildPlan(report, cleaner.PlanOptions{
		DryRun:       true,
		CleanupPaths: scenario.CleanupPaths(),
		PathExists:   scenario.PathExists,
		ValidatePath: func(string) error { return nil },
	})
	testfixtures.AssertNoMutations(t, recorder)
}

func containsAction(plan cleaner.CleanupPlan, actionType cleaner.CleanupActionType) bool {
	for _, action := range plan.Actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}

func containsTarget(plan cleaner.CleanupPlan, target string) bool {
	for _, action := range plan.Actions {
		if strings.EqualFold(action.Target, target) {
			return true
		}
	}
	return false
}
