package repair

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
)

func TestRepairDryRunBuildStatus(t *testing.T) {
	msi := tempMSI(t)
	resolver := func(string) (installer.InstallerResolution, error) {
		return installer.InstallerResolution{
			SelectedPath:   msi,
			SelectedSource: installer.InstallerSourceExplicit,
		}, nil
	}
	okCheck := func() (bool, error) { return true, nil }

	tests := []struct {
		name           string
		invite         string
		isAdmin        bool
		resolver       func(string) (installer.InstallerResolution, error)
		wantStatus     string
		wantReady      bool
		wantInput      string
		wantHasInvite  bool
		wantInstaller  bool
		wantAdminError bool
	}{
		{
			name:           "missing admin marks not ready",
			invite:         "SECRET-INVITE",
			isAdmin:        false,
			resolver:       resolver,
			wantStatus:     RepairPlanStatusNotReady,
			wantReady:      false,
			wantHasInvite:  true,
			wantInstaller:  true,
			wantAdminError: true,
		},
		{
			name:          "missing installer marks not ready",
			invite:        "SECRET-INVITE",
			isAdmin:       true,
			resolver:      missingInstallerResolver,
			wantStatus:    RepairPlanStatusNotReady,
			wantReady:     false,
			wantInput:     "installer",
			wantHasInvite: true,
		},
		{
			name:          "missing invite marks not ready",
			isAdmin:       true,
			resolver:      resolver,
			wantStatus:    RepairPlanStatusNotReady,
			wantReady:     false,
			wantInput:     "invite",
			wantInstaller: true,
		},
		{
			name:          "ready preflight marks planned",
			invite:        "SECRET-INVITE",
			isAdmin:       true,
			resolver:      resolver,
			wantStatus:    RepairPlanStatusPlanned,
			wantReady:     true,
			wantHasInvite: true,
			wantInstaller: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext(t)
			plan, _, err := DryRunWorkflow{
				Invite:            tt.invite,
				IsAdmin:           func() bool { return tt.isAdmin },
				Detect:            func() detector.DetectionReport { return healthyReport(nil) },
				BuildCleanupPlan:  emptyCleanupPlan,
				InstallerResolver: tt.resolver,
				PowerShellCheck:   okCheck,
				MSIExecCheck:      okCheck,
			}.Build(ctx)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if plan.Status != tt.wantStatus || plan.ReadyForRepair != tt.wantReady {
				t.Fatalf("status/ready = %s/%t, want %s/%t", plan.Status, plan.ReadyForRepair, tt.wantStatus, tt.wantReady)
			}
			if plan.Installer.HasInvite != tt.wantHasInvite {
				t.Fatalf("has invite = %t, want %t", plan.Installer.HasInvite, tt.wantHasInvite)
			}
			if plan.Installer.InstallerFound != tt.wantInstaller {
				t.Fatalf("installer found = %t, want %t", plan.Installer.InstallerFound, tt.wantInstaller)
			}
			if tt.wantInput != "" && !contains(plan.RequiredInputs, tt.wantInput) {
				t.Fatalf("required inputs = %#v, want %q", plan.RequiredInputs, tt.wantInput)
			}
			if tt.wantAdminError && !strings.Contains(strings.Join(plan.Errors, "\n"), "admin_rights") {
				t.Fatalf("errors = %#v, want admin_rights failure", plan.Errors)
			}
		})
	}
}

func TestRepairDryRunPlansWithoutMutating(t *testing.T) {
	ctx := newTestContext(t)
	msi := tempMSI(t)
	cleanupBuilt := false
	plan, _, err := DryRunWorkflow{
		Invite:  "SECRET-INVITE",
		IsAdmin: func() bool { return true },
		Detect: func() detector.DetectionReport {
			report := brokenReport()
			report.MissingDefenderPaths = []string{`C:\Program Files\TeleLinkSoft\bin`}
			report.Defender.ExclusionPaths = nil
			report.Defender.MissingPaths = report.MissingDefenderPaths
			return report
		},
		BuildCleanupPlan: func(detector.DetectionReport) cleaner.CleanupPlan {
			cleanupBuilt = true
			return cleaner.CleanupPlan{DryRun: true, Actions: []cleaner.CleanupAction{{
				Type:     cleaner.CleanupActionDeleteService,
				Target:   "ngs",
				Safe:     true,
				WouldRun: true,
			}}}
		},
		InstallerResolver: func(string) (installer.InstallerResolution, error) {
			return installer.InstallerResolution{SelectedPath: msi, SelectedSource: installer.InstallerSourceExplicit}, nil
		},
		PowerShellCheck: func() (bool, error) { return true, nil },
		MSIExecCheck:    func() (bool, error) { return true, nil },
	}.Build(ctx)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !cleanupBuilt {
		t.Fatal("cleanup plan was not generated")
	}
	if plan.Cleanup.ActionsCount != 1 {
		t.Fatalf("cleanup actions = %d, want 1", plan.Cleanup.ActionsCount)
	}
	if !plan.Defender.WouldAddExclusion {
		t.Fatal("Defender plan should add missing exclusion in real repair")
	}
	if !plan.Installer.WouldRunMsiexec {
		t.Fatal("installer plan should indicate msiexec would run in real repair")
	}
	if !strings.Contains(plan.Installer.Command, "invite=<REDACTED>") {
		t.Fatalf("installer command is not redacted: %q", plan.Installer.Command)
	}
}

func TestRepairDryRunRunWritesPlanOnlyArtifactsAndNoRawInvite(t *testing.T) {
	ctx := newTestContext(t)
	secret := "SECRET-INVITE-999"
	msi := filepath.Join(t.TempDir(), "Grabber Installer", "grabberEM.x64.msi")
	if err := os.MkdirAll(filepath.Dir(msi), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(msi, []byte("msi"), 0644); err != nil {
		t.Fatal(err)
	}
	err := DryRunWorkflow{
		Invite:           secret,
		IsAdmin:          func() bool { return true },
		Detect:           func() detector.DetectionReport { return healthyReport(nil) },
		BuildCleanupPlan: emptyCleanupPlan,
		InstallerResolver: func(string) (installer.InstallerResolution, error) {
			return installer.InstallerResolution{SelectedPath: msi, SelectedSource: installer.InstallerSourceExplicit}, nil
		},
		PowerShellCheck: func() (bool, error) { return true, nil },
		MSIExecCheck:    func() (bool, error) { return true, nil },
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, name := range []string{
		"repair-plan.json",
		"preflight-result.json",
		"initial-detection.json",
		"cleanup-plan.json",
		"classification-result.json",
		"recommendation-result.json",
		"operations.json",
		"summary.txt",
	} {
		assertExists(t, filepath.Join(ctx.OutputDir, name))
		assertFileDoesNotContain(t, filepath.Join(ctx.OutputDir, name), secret)
	}
	for _, name := range []string{"install-result.json", "defender-result.json", "final-detection.json"} {
		if _, err := os.Stat(filepath.Join(ctx.OutputDir, name)); err == nil {
			t.Fatalf("%s should not be written by repair --dry-run", name)
		}
	}
	planBytes, err := os.ReadFile(filepath.Join(ctx.OutputDir, "repair-plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan RepairPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.NextCommand, `--installer "`+msi+`"`) {
		t.Fatalf("next command does not quote installer path with spaces: %q", plan.NextCommand)
	}
	if !strings.Contains(plan.NextCommand, `--invite "<INVITE>"`) {
		t.Fatalf("next command does not use invite placeholder: %q", plan.NextCommand)
	}
}

func TestRepairDryRunExitCodesAndOutputs(t *testing.T) {
	if got := exitCodeForPlanStatus(RepairPlanStatusPlanned); got != app.ExitSuccess {
		t.Fatalf("planned exit = %d", got)
	}
	if got := exitCodeForPlanStatus(RepairPlanStatusPlannedWithWarnings); got != app.ExitWarnings {
		t.Fatalf("warnings exit = %d", got)
	}
	if got := exitCodeForPlanStatus(RepairPlanStatusNotReady); got != app.ExitVerificationFailed {
		t.Fatalf("not ready exit = %d", got)
	}
	outputs := expectedRepairOutputs(cleaner.CleanupPlan{Actions: []cleaner.CleanupAction{{Type: cleaner.CleanupActionMSIUninstall}}})
	for _, want := range []string{"final-detection.json", "install-result.json", "defender-result.json", "msi-uninstall.log"} {
		if !contains(outputs, want) {
			t.Fatalf("expected outputs = %#v, missing %q", outputs, want)
		}
	}
}

func missingInstallerResolver(string) (installer.InstallerResolution, error) {
	return installer.InstallerResolution{SelectedSource: installer.InstallerSourceNotFound}, installer.ErrInstallerNotFound
}

func emptyCleanupPlan(detector.DetectionReport) cleaner.CleanupPlan {
	return cleaner.CleanupPlan{DryRun: true}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s missing: %v", path, err)
	}
}

func assertFileDoesNotContain(t *testing.T, path string, value string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), value) {
		t.Fatalf("%s contains raw invite", path)
	}
}
