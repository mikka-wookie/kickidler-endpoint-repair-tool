package classifier_test

import (
	"testing"

	"kigrepair/internal/classifier"
	"kigrepair/internal/detector"
	"kigrepair/internal/verifier"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name        string
		input       classifier.ClassificationInput
		wantStatus  string
		wantPrimary string
		wantIssues  []string
	}{
		{
			name:        "healthy detection and passed verification",
			input:       classifier.ClassificationInput{Detection: ptrDetection(healthyDetection()), Verification: &verifier.VerificationResult{OverallStatus: verifier.VerificationPassed}},
			wantStatus:  classifier.StatusNoIssue,
			wantPrimary: classifier.CodeHealthy,
		},
		{
			name:        "not installed",
			input:       classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{Health: detector.GrabberHealthNotInstalled, InstallMode: detector.InstallModeNotInstalled})},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeNotInstalled,
		},
		{
			name:        "service executable missing",
			input:       classifier.ClassificationInput{Detection: ptrDetection(serviceBinaryMissingDetection(false))},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeServiceBinaryMissing,
		},
		{
			name:        "service executable missing and defender missing",
			input:       classifier.ClassificationInput{Detection: ptrDetection(serviceBinaryMissingDetection(true))},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeServiceBinaryMissing,
			wantIssues:  []string{classifier.CodeDefenderExclusionMissing},
		},
		{
			name:        "defender missing",
			input:       classifier.ClassificationInput{Detection: ptrDetection(defenderMissingDetection())},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeDefenderExclusionMissing,
		},
		{
			name: "msi registry leftovers",
			input: classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{
				Health:   detector.GrabberHealthPartiallyRemoved,
				Registry: []detector.RegistryState{{Root: "HKLM", Path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`, Exists: true}},
			})},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodePartialMSILeftovers,
		},
		{
			name: "files leftover",
			input: classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{
				Health: detector.GrabberHealthPartiallyRemoved,
				Files:  []detector.FileState{{Path: `C:\Program Files\TeleLinkSoft`, Exists: true, Type: "folder"}},
			})},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodePartialFilesLeftover,
		},
		{
			name: "wmi inconsistent",
			input: classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{
				Health:                  detector.GrabberHealthBroken,
				InstallMode:             detector.InstallModeHiddenWMI,
				InstallRoot:             `C:\Windows\System32\wmi`,
				PrimaryService:          "WmiProviderSE",
				ServiceExecutablePath:   `C:\Windows\System32\wmi\bin\svchost.exe`,
				ServiceExecutableExists: false,
				Services:                []detector.ServiceState{{Name: "WmiProviderSE", Exists: true, Status: "stopped", ImagePath: `C:\Windows\System32\wmi\bin\svchost.exe`, ExpectedImagePathMatch: true}},
			})},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeWMIHiddenModeInconsistent,
		},
		{
			name: "service stopped",
			input: classifier.ClassificationInput{Detection: ptrDetection(stoppedServiceDetection()), Verification: &verifier.VerificationResult{
				OverallStatus: verifier.VerificationWarning,
				Checks:        []verifier.VerificationCheck{{Name: "primary_service_running", Status: verifier.VerificationWarning, Message: "Service exists and executable exists, but service is not running"}},
			}},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeServiceNotRunning,
		},
		{
			name:        "verification failed fallback",
			input:       classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{Health: detector.GrabberHealthHealthy, InstallMode: detector.InstallModeStandard}), Verification: &verifier.VerificationResult{OverallStatus: verifier.VerificationFailed}},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeVerificationFailed,
		},
		{
			name:        "verification warning fallback",
			input:       classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{Health: detector.GrabberHealthHealthy, InstallMode: detector.InstallModeStandard}), Verification: &verifier.VerificationResult{OverallStatus: verifier.VerificationWarning}},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeVerificationWarning,
		},
		{
			name: "defender query failed",
			input: classifier.ClassificationInput{Detection: ptrDetection(detector.DetectionReport{
				Health:                detector.GrabberHealthHealthy,
				InstallMode:           detector.InstallModeStandard,
				InstallRoot:           `C:\Program Files\TeleLinkSoft`,
				RequiredDefenderPaths: []string{`C:\Program Files\TeleLinkSoft`},
				Defender:              detector.DefenderState{Available: false, Error: "access denied"},
			})},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeDefenderStatusUnavailable,
		},
		{
			name:        "nil input",
			input:       classifier.ClassificationInput{},
			wantStatus:  classifier.StatusInconclusive,
			wantPrimary: classifier.CodeUnknownInstallState,
		},
		{
			name:        "priority is stable",
			input:       classifier.ClassificationInput{Detection: ptrDetection(serviceBinaryMissingDetection(true)), Verification: &verifier.VerificationResult{OverallStatus: verifier.VerificationFailed}},
			wantStatus:  classifier.StatusIssueDetected,
			wantPrimary: classifier.CodeServiceBinaryMissing,
			wantIssues:  []string{classifier.CodeDefenderExclusionMissing, classifier.CodeVerificationFailed},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.Classify(tt.input)
			if got.Status != tt.wantStatus {
				t.Fatalf("Status = %s, want %s: %#v", got.Status, tt.wantStatus, got)
			}
			if got.PrimaryIssue == nil || got.PrimaryIssue.Code != tt.wantPrimary {
				t.Fatalf("PrimaryIssue = %#v, want %s", got.PrimaryIssue, tt.wantPrimary)
			}
			for _, code := range tt.wantIssues {
				if !hasIssue(got.Issues, code) {
					t.Fatalf("issues missing %s: %#v", code, got.Issues)
				}
			}
			if got.RecommendedAction == "" || got.SupportSummary == "" {
				t.Fatalf("classification missing action or summary: %#v", got)
			}
		})
	}
}

func ptrDetection(report detector.DetectionReport) *detector.DetectionReport {
	return &report
}

func healthyDetection() detector.DetectionReport {
	return detector.DetectionReport{
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft`,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\grabber2.exe`,
		ServiceExecutableExists: true,
		RequiredDefenderPaths:   []string{`C:\Program Files\TeleLinkSoft`},
		Services:                []detector.ServiceState{{Name: "ngs", Exists: true, Status: "running", ImagePath: `C:\Program Files\TeleLinkSoft\grabber2.exe`}},
		Defender:                detector.DefenderState{Available: true, CoveredPaths: []string{`C:\Program Files\TeleLinkSoft`}},
	}
}

func serviceBinaryMissingDetection(defenderMissing bool) detector.DetectionReport {
	report := healthyDetection()
	report.Health = detector.GrabberHealthBroken
	report.ServiceExecutableExists = false
	report.Services[0].Status = "running"
	if defenderMissing {
		report.Defender.Available = true
		report.MissingDefenderPaths = []string{report.InstallRoot}
		report.Defender.MissingPaths = []string{report.InstallRoot}
		report.Defender.CoveredPaths = nil
	}
	return report
}

func defenderMissingDetection() detector.DetectionReport {
	report := healthyDetection()
	report.MissingDefenderPaths = []string{report.InstallRoot}
	report.Defender.MissingPaths = []string{report.InstallRoot}
	report.Defender.CoveredPaths = nil
	return report
}

func stoppedServiceDetection() detector.DetectionReport {
	report := healthyDetection()
	report.Health = detector.GrabberHealthBroken
	report.Services[0].Status = "stopped"
	return report
}

func hasIssue(issues []classifier.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
