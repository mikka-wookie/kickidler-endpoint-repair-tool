package recommendations

import (
	"strings"
	"testing"

	"kigrepair/internal/detector"
)

func TestPlanRequiredCases(t *testing.T) {
	tests := []struct {
		name          string
		input         RecommendationInput
		wantPrimary   string
		wantStatus    string
		wantInputs    []string
		wantSecondary string
	}{
		{
			name: "healthy classification",
			input: RecommendationInput{Classification: &ClassificationResult{
				PrimaryIssue: &ClassifiedIssue{Code: issueHealthy},
			}, IsAdmin: true},
			wantPrimary: ActionNoRepairRequired,
			wantStatus:  StatusNoActionRequired,
		},
		{
			name:        "not installed missing installer invite",
			input:       RecommendationInput{Detection: detection(detector.GrabberHealthNotInstalled), IsAdmin: true},
			wantPrimary: ActionInstallOrRepairWithInvite,
			wantStatus:  StatusMissingRequiredInput,
			wantInputs:  []string{InputInstaller, InputInvite},
		},
		{
			name: "service binary missing with inputs",
			input: RecommendationInput{
				Detection:     brokenDetection(),
				HasInstaller:  true,
				HasInvite:     true,
				InstallerPath: `C:\Installers\grabberEM.x64.msi`,
				IsAdmin:       true,
			},
			wantPrimary: ActionRunFullRepair,
			wantStatus:  StatusActionRecommended,
		},
		{
			name:        "service binary missing invite absent",
			input:       RecommendationInput{Detection: brokenDetection(), HasInstaller: true, IsAdmin: true},
			wantPrimary: ActionRunFullRepair,
			wantStatus:  StatusMissingRequiredInput,
			wantInputs:  []string{InputInvite},
		},
		{
			name:        "defender missing only",
			input:       RecommendationInput{Detection: defenderMissingHealthy(), IsAdmin: true},
			wantPrimary: ActionRunDefenderEnsure,
			wantStatus:  StatusActionRecommended,
		},
		{
			name:          "partial msi leftovers",
			input:         RecommendationInput{Detection: partialMSI(), IsAdmin: true},
			wantPrimary:   ActionRunCleanupDryRun,
			wantStatus:    StatusActionRecommended,
			wantSecondary: ActionRunCleanupThenRepair,
		},
		{
			name:          "partial files leftover",
			input:         RecommendationInput{Detection: detection(detector.GrabberHealthPartiallyRemoved), IsAdmin: true},
			wantPrimary:   ActionRunCleanupDryRun,
			wantStatus:    StatusActionRecommended,
			wantSecondary: ActionRunCleanupThenRepair,
		},
		{
			name:        "unknown install state",
			input:       RecommendationInput{Detection: detection(detector.GrabberHealthUnknown), IsAdmin: true},
			wantPrimary: ActionCollectSupportBundle,
			wantStatus:  StatusEscalationRecommended,
		},
		{
			name:        "verification warning",
			input:       RecommendationInput{Verification: &VerificationState{OverallStatus: "warning"}, IsAdmin: true},
			wantPrimary: ActionCollectSupportBundle,
			wantStatus:  StatusActionRecommended,
		},
		{
			name:        "repair failed verification failed",
			input:       RecommendationInput{Verification: &VerificationState{OverallStatus: "failed"}, RepairFailed: true, IsAdmin: true},
			wantPrimary: ActionEscalateWithBundle,
			wantStatus:  StatusEscalationRecommended,
		},
		{
			name:        "nil classification does not panic",
			input:       RecommendationInput{IsAdmin: true},
			wantPrimary: ActionCollectSupportBundle,
			wantStatus:  StatusEscalationRecommended,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Plan(tt.input)
			if got.PrimaryAction == nil {
				t.Fatal("PrimaryAction is nil")
			}
			if got.PrimaryAction.Code != tt.wantPrimary {
				t.Fatalf("primary = %s, want %s", got.PrimaryAction.Code, tt.wantPrimary)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %s, want %s", got.Status, tt.wantStatus)
			}
			for _, input := range tt.wantInputs {
				if !hasRequiredInput(got.RequiredInputs, input) {
					t.Fatalf("missing required input %s in %#v", input, got.RequiredInputs)
				}
			}
			if tt.wantSecondary != "" && (len(got.Actions) < 2 || got.Actions[1].Code != tt.wantSecondary) {
				t.Fatalf("secondary = %#v, want %s", got.Actions, tt.wantSecondary)
			}
		})
	}
}

func TestCommandBuilderQuotesInstallerWithSpaces(t *testing.T) {
	got := RepairCommand(`C:\Installers With Spaces\grabberEM.x64.msi`)
	want := `--installer "C:\Installers With Spaces\grabberEM.x64.msi"`
	if !strings.Contains(got, want) {
		t.Fatalf("command %q does not contain %q", got, want)
	}
}

func TestCommandBuilderNeverOutputsRawInvite(t *testing.T) {
	rawInvite := "secret-invite-value"
	got := RepairCommand(`C:\Installers\grabberEM.x64.msi`)
	if strings.Contains(got, rawInvite) {
		t.Fatalf("command leaked raw invite")
	}
	if !strings.Contains(got, "<INVITE>") {
		t.Fatalf("command does not contain invite placeholder: %s", got)
	}
}

func TestActionPriorityStable(t *testing.T) {
	got := Plan(RecommendationInput{Detection: partialMSI(), IsAdmin: true})
	if len(got.Actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(got.Actions))
	}
	if got.Actions[0].Priority >= got.Actions[1].Priority {
		t.Fatalf("priorities should be stable ascending: %#v", got.Actions)
	}
}

func detection(health detector.GrabberHealthStatus) *detector.DetectionReport {
	return &detector.DetectionReport{Health: health, Defender: detector.DefenderState{Available: true}}
}

func brokenDetection() *detector.DetectionReport {
	return &detector.DetectionReport{
		Health:                detector.GrabberHealthBroken,
		InstallMode:           detector.InstallModeStandard,
		ServiceExecutablePath: `C:\Program Files\TeleLinkSoft\grabber2.exe`,
		Defender:              detector.DefenderState{Available: true},
	}
}

func defenderMissingHealthy() *detector.DetectionReport {
	return &detector.DetectionReport{
		Health:               detector.GrabberHealthHealthy,
		InstallRoot:          `C:\Program Files\TeleLinkSoft`,
		MissingDefenderPaths: []string{`C:\Program Files\TeleLinkSoft`},
		Defender:             detector.DefenderState{Available: true},
	}
}

func partialMSI() *detector.DetectionReport {
	report := detection(detector.GrabberHealthPartiallyRemoved)
	report.Registry = []detector.RegistryState{{Path: `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EB1FBC37-0B97-4CF5-A329-CF28BA653748}`, Exists: true}}
	return report
}

func hasRequiredInput(inputs []RequiredInput, name string) bool {
	for _, input := range inputs {
		if input.Name == name {
			return true
		}
	}
	return false
}
