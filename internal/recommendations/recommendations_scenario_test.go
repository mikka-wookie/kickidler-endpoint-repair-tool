package recommendations_test

import (
	"testing"

	"kigrepair/internal/classifier"
	"kigrepair/internal/recommendations"
	"kigrepair/internal/testfixtures"
)

func TestRecommendationScenarios(t *testing.T) {
	for _, name := range []string{
		"healthy_standard_install",
		"defender_deleted_service_binary",
		"not_installed_clean_machine",
		"partial_msi_leftovers",
		"partial_files_leftover",
		"hidden_wmi_healthy",
		"hidden_wmi_service_binary_missing",
		"supported_service_name_path_mismatch",
		"defender_query_failed",
	} {
		t.Run(name, func(t *testing.T) {
			scenario := testfixtures.ScenarioByName(name)
			report := scenario.DetectionReport()
			classification := classifier.Classify(classifier.ClassificationInput{Detection: &report})
			converted := recommendations.FromClassifier(classification)
			result := recommendations.Plan(recommendations.RecommendationInput{
				Detection:      &report,
				Classification: &converted,
				HasInstaller:   scenario.Installer.Valid,
				HasInvite:      true,
				IsAdmin:        scenario.IsAdmin,
				IsInteractive:  false,
				OutputDir:      `C:\ProgramData\kigrepair\Reports\fixture`,
			})
			if result.PrimaryAction == nil {
				t.Fatal("primary action is nil")
			}
			if scenario.Expected.RecommendationCode != "" && result.PrimaryAction.Code != scenario.Expected.RecommendationCode {
				t.Fatalf("primary action = %s, want %s; actions=%#v", result.PrimaryAction.Code, scenario.Expected.RecommendationCode, result.Actions)
			}
		})
	}
}

func TestMissingInviteRecommendationDoesNotExposeSecret(t *testing.T) {
	scenario := testfixtures.ScenarioByName("missing_invite_blocks_repair_readiness")
	report := scenario.DetectionReport()
	classification := classifier.Classify(classifier.ClassificationInput{Detection: &report})
	converted := recommendations.FromClassifier(classification)
	result := recommendations.Plan(recommendations.RecommendationInput{
		Detection:      &report,
		Classification: &converted,
		HasInstaller:   true,
		HasInvite:      false,
		IsAdmin:        true,
	})
	if result.Status != recommendations.StatusMissingRequiredInput {
		t.Fatalf("status = %s, want %s", result.Status, recommendations.StatusMissingRequiredInput)
	}
	if !requiresInput(result, recommendations.InputInvite) {
		t.Fatalf("required inputs = %#v, want invite", result.RequiredInputs)
	}
}

func requiresInput(result recommendations.RecommendationResult, name string) bool {
	for _, input := range result.RequiredInputs {
		if input.Name == name {
			return true
		}
	}
	return false
}
