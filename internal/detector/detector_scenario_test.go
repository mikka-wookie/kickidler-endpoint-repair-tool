package detector_test

import (
	"testing"

	"kigrepair/internal/detector"
	"kigrepair/internal/testfixtures"
)

func TestDetectorScenarios(t *testing.T) {
	for _, name := range []string{
		"healthy_standard_install",
		"defender_deleted_service_binary",
		"not_installed_clean_machine",
		"partial_msi_leftovers",
		"partial_files_leftover",
		"hidden_wmi_healthy",
		"hidden_wmi_service_binary_missing",
		"normal_windows_svchost_not_grabber",
		"supported_service_name_path_mismatch",
		"process_name_match_path_mismatch",
	} {
		t.Run(name, func(t *testing.T) {
			scenario := testfixtures.ScenarioByName(name)
			report := scenario.DetectionReport()
			if scenario.Expected.Health != "" && string(report.Health) != scenario.Expected.Health {
				t.Fatalf("health = %s, want %s; issues=%v", report.Health, scenario.Expected.Health, report.Issues)
			}
			if scenario.Expected.InstallMode != "" && string(report.InstallMode) != scenario.Expected.InstallMode {
				t.Fatalf("install mode = %s, want %s", report.InstallMode, scenario.Expected.InstallMode)
			}
			if scenario.Expected.InstallRoot != "" && !pathsEqual(report.InstallRoot, scenario.Expected.InstallRoot) {
				t.Fatalf("install root = %s, want %s", report.InstallRoot, scenario.Expected.InstallRoot)
			}
		})
	}
}

func TestNormalWindowsSvchostDoesNotInferHiddenWMI(t *testing.T) {
	report := testfixtures.ScenarioByName("normal_windows_svchost_not_grabber").DetectionReport()
	if report.InstallMode == detector.InstallModeHiddenWMI {
		t.Fatal("normal Windows svchost inferred hidden WMI mode")
	}
	for _, process := range report.Processes {
		if process.CanTerminate {
			t.Fatalf("normal Windows process is terminable: %#v", process)
		}
	}
}

func TestDefenderCoverageScenarios(t *testing.T) {
	tests := []string{
		"defender_parent_exclusion_covers_child",
		"defender_prefix_false_positive",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			scenario := testfixtures.ScenarioByName(name)
			required := scenario.Defender.RequiredPaths[0]
			covered := detector.IsPathCoveredByAnyExclusion(required, scenario.Defender.ExclusionPaths)
			if covered != scenario.Expected.DefenderCovered {
				t.Fatalf("covered = %t, want %t", covered, scenario.Expected.DefenderCovered)
			}
		})
	}
}

func pathsEqual(left string, right string) bool {
	return detector.NormalizeWindowsPath(left) == detector.NormalizeWindowsPath(right)
}
