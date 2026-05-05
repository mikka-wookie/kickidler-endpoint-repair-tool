package wizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/repair"
	"kigrepair/internal/reports"
	"kigrepair/internal/verifier"
)

func TestWizardHealthyStateCompletesWithoutRepair(t *testing.T) {
	ctx := testContext(t)
	repairCalled := false
	workflow := Workflow{
		Options: Options{NonInteractive: true},
		Detect:  func() detector.DetectionReport { return healthyDetection() },
		RunRepair: func(ctx *app.AppContext, opts Options) error {
			repairCalled = true
			return nil
		},
		RunVerify:  fakeVerify(verifier.VerificationPassed),
		RunCollect: fakeCollect(t),
	}

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	result := ctx.JSONValue.(Result)
	if repairCalled {
		t.Fatal("repair was called for healthy state")
	}
	if result.Status != StatusCompleted {
		t.Fatalf("status = %q, want %q", result.Status, StatusCompleted)
	}
	if result.RepairExecuted {
		t.Fatal("repair_executed = true")
	}
}

func TestWizardDeclinedRepairDoesNotMutate(t *testing.T) {
	ctx := testContext(t)
	repairCalled := false
	workflow := Workflow{
		Options:       Options{AllowRepair: true, InstallerPath: `C:\grabber.msi`, InviteValue: "SECRET-INVITE", HasInvite: true},
		Prompter:      &FakePrompter{Strings: []string{"NO"}},
		Detect:        func() detector.DetectionReport { return brokenDetection() },
		BuildDryRun:   readyPlan,
		RunVerify:     fakeVerify(verifier.VerificationWarning),
		RunCollect:    fakeCollect(t),
		ConfirmRepair: func(string) bool { return false },
		RunRepair: func(ctx *app.AppContext, opts Options) error {
			repairCalled = true
			return nil
		},
	}

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	result := ctx.JSONValue.(Result)
	if repairCalled {
		t.Fatal("repair was called after declined confirmation")
	}
	if result.Status != StatusCancelled {
		t.Fatalf("status = %q, want %q", result.Status, StatusCancelled)
	}
}

func TestWizardConfirmedReadyRepairCallsRepair(t *testing.T) {
	ctx := testContext(t)
	repairCalled := false
	workflow := Workflow{
		Options:       Options{AllowRepair: true, InstallerPath: `C:\grabber.msi`, InviteValue: "SECRET-INVITE", HasInvite: true},
		Detect:        func() detector.DetectionReport { return brokenDetection() },
		BuildDryRun:   readyPlan,
		RunVerify:     fakeVerify(verifier.VerificationPassed),
		RunCollect:    fakeCollect(t),
		ConfirmRepair: func(string) bool { return true },
		RunRepair: func(ctx *app.AppContext, opts Options) error {
			repairCalled = true
			if opts.InviteValue != "SECRET-INVITE" {
				t.Fatalf("invite was not passed internally")
			}
			return nil
		},
	}

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	result := ctx.JSONValue.(Result)
	if !repairCalled {
		t.Fatal("repair was not called")
	}
	if result.Status != StatusRepairCompleted {
		t.Fatalf("status = %q, want %q", result.Status, StatusRepairCompleted)
	}
}

func TestWizardNonInteractiveRepairRequiresYes(t *testing.T) {
	ctx := testContext(t)
	repairCalled := false
	workflow := Workflow{
		Options:     Options{NonInteractive: true, AllowRepair: true, InstallerPath: `C:\grabber.msi`, InviteValue: "SECRET-INVITE", HasInvite: true},
		Detect:      func() detector.DetectionReport { return brokenDetection() },
		BuildDryRun: readyPlan,
		RunVerify:   fakeVerify(verifier.VerificationWarning),
		RunCollect:  fakeCollect(t),
		RunRepair: func(ctx *app.AppContext, opts Options) error {
			repairCalled = true
			return nil
		},
	}
	ctx.NonInteractive = true

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if repairCalled {
		t.Fatal("repair was called without --yes")
	}
	result := ctx.JSONValue.(Result)
	if result.Status != StatusCancelled {
		t.Fatalf("status = %q, want %q", result.Status, StatusCancelled)
	}
}

func TestWizardDoesNotWriteRawInvite(t *testing.T) {
	ctx := testContext(t)
	workflow := Workflow{
		Options:       Options{AllowRepair: true, InstallerPath: `C:\grabber.msi`, InviteValue: "RAW-SECRET-INVITE", HasInvite: true},
		Detect:        func() detector.DetectionReport { return brokenDetection() },
		BuildDryRun:   readyPlan,
		RunVerify:     fakeVerify(verifier.VerificationPassed),
		RunCollect:    fakeCollect(t),
		ConfirmRepair: func(string) bool { return true },
		RunRepair:     func(ctx *app.AppContext, opts Options) error { return nil },
	}

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, name := range []string{"wizard-result.json", "summary.txt", "operations.json"} {
		data, err := os.ReadFile(filepath.Join(ctx.OutputDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(data), "RAW-SECRET-INVITE") {
			t.Fatalf("%s contains raw invite", name)
		}
	}
}

func TestWizardCollectBundleIncludesWizardResult(t *testing.T) {
	ctx := testContext(t)
	workflow := Workflow{
		Options: Options{NonInteractive: true, CollectBundle: true},
		Detect:  func() detector.DetectionReport { return healthyDetection() },
		RunVerify: func(ctx *app.AppContext) (*verifier.VerificationResult, error) {
			return &verifier.VerificationResult{OverallStatus: verifier.VerificationPassed}, nil
		},
	}
	ctx.NonInteractive = true

	if err := workflow.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	archive := reports.InspectSupportBundle(ctx.OutputDir)
	found := false
	for _, file := range archive.Files {
		if file == "reports/wizard-result.json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("support bundle does not include wizard-result.json; files=%v", archive.Files)
	}
}

func testContext(t *testing.T) *app.AppContext {
	t.Helper()
	dir := t.TempDir()
	reporter, err := reports.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger, err := logging.New(filepath.Join(dir, "repair.log"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	ctx := app.NewContext()
	ctx.OutputDir = dir
	ctx.Reporter = reporter
	ctx.Logger = logger
	ctx.Quiet = true
	return ctx
}

func healthyDetection() detector.DetectionReport {
	return detector.DetectionReport{
		IsAdmin:                 true,
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft`,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutableExists: true,
		Services: []detector.ServiceState{{
			Name:   "ngs",
			Exists: true,
			Status: "running",
		}},
		RequiredDefenderPaths: []string{`C:\Program Files\TeleLinkSoft`},
		Defender: detector.DefenderState{
			Available:      true,
			ExclusionPaths: []string{`C:\Program Files\TeleLinkSoft`},
		},
	}
}

func brokenDetection() detector.DetectionReport {
	return detector.DetectionReport{
		IsAdmin:                 true,
		Health:                  detector.GrabberHealthBroken,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             `C:\Program Files\TeleLinkSoft`,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   `C:\Program Files\TeleLinkSoft\bin\grabber2.exe`,
		ServiceExecutableExists: false,
		Services: []detector.ServiceState{{
			Name:   "ngs",
			Exists: true,
			Status: "stopped",
		}},
		Issues: []string{"Missing required file: C:\\Program Files\\TeleLinkSoft\\bin\\grabber2.exe"},
	}
}

func readyPlan(ctx *app.AppContext, opts Options) (repair.RepairPlan, error) {
	return repair.RepairPlan{
		Status:         repair.RepairPlanStatusPlanned,
		ReadyForRepair: true,
		ReportDir:      ctx.OutputDir,
		Preflight: &repair.PreflightResult{
			AdminRights:         true,
			InstallerAvailable:  true,
			InvitePresent:       true,
			MSIExecAvailable:    true,
			PowerShellAvailable: true,
			Checks: []repair.PreflightCheck{
				{Name: "admin_rights", Status: "pass", Required: true},
				{Name: "installer_available", Status: "pass", Required: true},
				{Name: "invite_present", Status: "pass", Required: true},
			},
		},
	}, nil
}

func fakeVerify(status verifier.VerificationStatus) func(*app.AppContext) (*verifier.VerificationResult, error) {
	return func(ctx *app.AppContext) (*verifier.VerificationResult, error) {
		return &verifier.VerificationResult{OverallStatus: status}, nil
	}
}

func fakeCollect(t *testing.T) func(*app.AppContext) error {
	t.Helper()
	return func(ctx *app.AppContext) error {
		if err := ctx.Reporter.WriteJSON("collect-result", map[string]string{"status": "success"}); err != nil {
			return err
		}
		_, err := reports.CreateSupportBundle(ctx.OutputDir)
		return err
	}
}
