package classifier_test

import (
	"testing"

	"kigrepair/internal/classifier"
	"kigrepair/internal/testfixtures"
)

func TestClassifierScenarios(t *testing.T) {
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
			result := classifier.Classify(classifier.ClassificationInput{Detection: &report})
			if result.PrimaryIssue == nil {
				t.Fatal("primary issue is nil")
			}
			if scenario.Expected.PrimaryIssueCode != "" && result.PrimaryIssue.Code != scenario.Expected.PrimaryIssueCode {
				t.Fatalf("primary issue = %s, want %s; issues=%#v", result.PrimaryIssue.Code, scenario.Expected.PrimaryIssueCode, result.Issues)
			}
		})
	}
}

func TestDefenderDeletedBinaryIncludesDefenderMissingIssue(t *testing.T) {
	report := testfixtures.ScenarioByName("defender_deleted_service_binary").DetectionReport()
	result := classifier.Classify(classifier.ClassificationInput{Detection: &report})
	if !hasScenarioIssue(result.Issues, classifier.CodeDefenderExclusionMissing) {
		t.Fatalf("issues = %#v, want %s", result.Issues, classifier.CodeDefenderExclusionMissing)
	}
}

func hasScenarioIssue(issues []classifier.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
