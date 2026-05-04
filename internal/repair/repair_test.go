package repair

import (
	"os"
	"path/filepath"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/defender"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
	"kigrepair/internal/logging"
	"kigrepair/internal/reports"
)

type fakeCleanupExecutor struct {
	results []app.OperationResult
	ran     bool
}

func (e *fakeCleanupExecutor) ExecutePlan(cleaner.CleanupPlan) []app.OperationResult {
	e.ran = true
	return e.results
}

type fakeMSIExecutor struct {
	code int
	ran  bool
}

func (e *fakeMSIExecutor) Install(installerPath string, invite string, logPath string) installer.CommandResult {
	e.ran = true
	return installer.CommandResult{ExitCode: e.code}
}

type fakeDefenderAdder struct {
	code int
	ran  bool
}

func (a *fakeDefenderAdder) AddExclusion(path string) defender.CommandResult {
	a.ran = true
	return defender.CommandResult{ExitCode: a.code}
}

func TestRepairWorkflowResults(t *testing.T) {
	tests := []struct {
		name          string
		initial       detector.DetectionReport
		sequence      []detector.DetectionReport
		cleanupResult []app.OperationResult
		msiCode       int
		adderCode     int
		force         bool
		wantExit      int
		wantCleanup   bool
		wantInstall   bool
	}{
		{
			name:        "healthy no force skips cleanup and install",
			initial:     healthyReport(nil),
			sequence:    []detector.DetectionReport{healthyReport(nil)},
			msiCode:     0,
			adderCode:   0,
			wantExit:    ExitSuccess,
			wantCleanup: false,
			wantInstall: false,
		},
		{
			name:        "healthy force runs cleanup and install",
			initial:     healthyReport(nil),
			sequence:    []detector.DetectionReport{healthyReport(nil), healthyReport(nil), healthyReport(nil)},
			msiCode:     0,
			adderCode:   0,
			force:       true,
			wantExit:    ExitSuccess,
			wantCleanup: true,
			wantInstall: true,
		},
		{
			name:        "install failed",
			initial:     brokenReport(),
			sequence:    []detector.DetectionReport{brokenReport()},
			msiCode:     1603,
			adderCode:   0,
			wantExit:    ExitInstallFailed,
			wantCleanup: true,
			wantInstall: true,
		},
		{
			name:        "final broken",
			initial:     brokenReport(),
			sequence:    []detector.DetectionReport{healthyReport(nil), brokenReport()},
			msiCode:     0,
			adderCode:   0,
			wantExit:    ExitVerificationFailed,
			wantCleanup: true,
			wantInstall: true,
		},
		{
			name:        "defender failed final healthy",
			initial:     brokenReport(),
			sequence:    []detector.DetectionReport{healthyReport([]string{`C:\Program Files\TeleLinkSoft\bin`}), healthyReport(nil), healthyReport(nil)},
			msiCode:     0,
			adderCode:   1,
			wantExit:    ExitWarnings,
			wantCleanup: true,
			wantInstall: true,
		},
		{
			name:    "cleanup failure final healthy",
			initial: brokenReport(),
			sequence: []detector.DetectionReport{
				healthyReport(nil),
				healthyReport(nil),
			},
			cleanupResult: []app.OperationResult{{
				Step:    string(cleaner.CleanupActionDeletePath),
				Target:  `C:\Program Files\TeleLinkSoft`,
				Status:  app.OperationStatusFailed,
				Message: "Path deletion failed",
				Error:   "access denied",
			}},
			msiCode:     0,
			adderCode:   0,
			wantExit:    ExitWarnings,
			wantCleanup: true,
			wantInstall: true,
		},
		{
			name:        "reboot required final healthy",
			initial:     brokenReport(),
			sequence:    []detector.DetectionReport{healthyReport(nil), healthyReport(nil)},
			msiCode:     3010,
			adderCode:   0,
			wantExit:    ExitRebootRequired,
			wantCleanup: true,
			wantInstall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installerPath := tempMSI(t)
			ctx := newTestContext(t)
			ctx.Force = tt.force
			cleanupExec := &fakeCleanupExecutor{results: tt.cleanupResult}
			if cleanupExec.results == nil {
				cleanupExec.results = []app.OperationResult{{
					Step:    string(cleaner.CleanupActionDeleteService),
					Target:  "ngs",
					Status:  app.OperationStatusSuccess,
					Message: "Service deleted",
				}}
			}
			msiExec := &fakeMSIExecutor{code: tt.msiCode}
			adder := &fakeDefenderAdder{code: tt.adderCode}
			detect := sequenceDetector(append([]detector.DetectionReport{tt.initial}, tt.sequence...))

			err := RepairWorkflow{
				Invite:    "abc123",
				Installer: installerPath,
				Yes:       true,
				IsAdmin:   func() bool { return true },
				Detect:    detect,
				BuildCleanupPlan: func(detector.DetectionReport) cleaner.CleanupPlan {
					return cleaner.CleanupPlan{Actions: []cleaner.CleanupAction{{Type: cleaner.CleanupActionDeleteService, Target: "ngs", Safe: true}}}
				},
				CleanupExecutor: cleanupExec,
				MSIExecutor:     msiExec,
				DefenderAdder:   adder,
			}.Run(ctx)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if ctx.ExitCode != tt.wantExit {
				t.Fatalf("ExitCode = %d, want %d", ctx.ExitCode, tt.wantExit)
			}
			if cleanupExec.ran != tt.wantCleanup {
				t.Fatalf("cleanup ran = %t, want %t", cleanupExec.ran, tt.wantCleanup)
			}
			if msiExec.ran != tt.wantInstall {
				t.Fatalf("install ran = %t, want %t", msiExec.ran, tt.wantInstall)
			}
		})
	}
}

func TestRepairWorkflowInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		invite    string
		installer string
	}{
		{name: "missing invite", installer: tempMSIName(t)},
		{name: "invalid invite", invite: "bad&invite", installer: tempMSIName(t)},
		{name: "missing installer", invite: "abc123"},
		{name: "non msi installer", invite: "abc123", installer: tempTextFile(t)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext(t)
			err := RepairWorkflow{
				Invite:    tt.invite,
				Installer: tt.installer,
				Yes:       true,
				IsAdmin:   func() bool { return true },
			}.Run(ctx)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if ctx.ExitCode != ExitInvalidInput {
				t.Fatalf("ExitCode = %d, want %d", ctx.ExitCode, ExitInvalidInput)
			}
		})
	}
}

func TestRepairWorkflowConfirmationRules(t *testing.T) {
	t.Run("non interactive without yes exits 3", func(t *testing.T) {
		ctx := newTestContext(t)
		ctx.NonInteractive = true
		err := RepairWorkflow{
			Invite:           "abc123",
			Installer:        tempMSI(t),
			IsAdmin:          func() bool { return true },
			Detect:           func() detector.DetectionReport { return healthyReport(nil) },
			BuildCleanupPlan: func(detector.DetectionReport) cleaner.CleanupPlan { return cleaner.CleanupPlan{} },
		}.Run(ctx)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if ctx.ExitCode != ExitConfirmationRequired {
			t.Fatalf("ExitCode = %d, want %d", ctx.ExitCode, ExitConfirmationRequired)
		}
	})

	t.Run("non interactive with yes proceeds", func(t *testing.T) {
		ctx := newTestContext(t)
		ctx.NonInteractive = true
		err := RepairWorkflow{
			Invite:           "abc123",
			Installer:        tempMSI(t),
			Yes:              true,
			IsAdmin:          func() bool { return true },
			Detect:           sequenceDetector([]detector.DetectionReport{healthyReport(nil), healthyReport(nil)}),
			BuildCleanupPlan: func(detector.DetectionReport) cleaner.CleanupPlan { return cleaner.CleanupPlan{} },
		}.Run(ctx)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if ctx.ExitCode != ExitSuccess {
			t.Fatalf("ExitCode = %d, want %d", ctx.ExitCode, ExitSuccess)
		}
	})
}

func TestFinalExitCode(t *testing.T) {
	if got := finalExitCode(installer.MSIResult{Success: false}, Verification{Status: "success"}, nil, false); got != ExitInstallFailed {
		t.Fatalf("install failed exit = %d", got)
	}
	if got := finalExitCode(installer.MSIResult{Success: true}, Verification{Status: "failed"}, nil, false); got != ExitVerificationFailed {
		t.Fatalf("verification failed exit = %d", got)
	}
	if got := finalExitCode(installer.MSIResult{Success: true, RebootRequired: true}, Verification{Status: "success"}, nil, false); got != ExitRebootRequired {
		t.Fatalf("reboot exit = %d", got)
	}
	if got := finalExitCode(installer.MSIResult{Success: true}, Verification{Status: "warning"}, nil, false); got != ExitWarnings {
		t.Fatalf("warning exit = %d", got)
	}
	if got := finalExitCode(installer.MSIResult{Success: true}, Verification{Status: "success"}, []app.OperationResult{{Step: "defender_ensure", Status: app.OperationStatusFailed}}, true); got != ExitWarnings {
		t.Fatalf("defender failed exit = %d", got)
	}
}

func newTestContext(t *testing.T) *app.AppContext {
	t.Helper()
	dir := t.TempDir()
	reporter, err := reports.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := app.NewContext()
	ctx.OutputDir = dir
	ctx.Reporter = reporter
	ctx.Logger = logging.Discard()
	ctx.Quiet = true
	return ctx
}

func tempMSI(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "grabber.msi")
	if err := os.WriteFile(path, []byte("msi"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func tempMSIName(t *testing.T) string {
	t.Helper()
	return tempMSI(t)
}

func tempTextFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "grabber.txt")
	if err := os.WriteFile(path, []byte("not msi"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func sequenceDetector(reports []detector.DetectionReport) func() detector.DetectionReport {
	index := 0
	return func() detector.DetectionReport {
		if index >= len(reports) {
			return reports[len(reports)-1]
		}
		report := reports[index]
		index++
		return report
	}
}

func healthyReport(missingDefender []string) detector.DetectionReport {
	root := `C:\Program Files\TeleLinkSoft\bin`
	return detector.DetectionReport{
		Health:                  detector.GrabberHealthHealthy,
		InstallMode:             detector.InstallModeStandard,
		InstallRoot:             root,
		PrimaryService:          "ngs",
		ServiceExecutablePath:   root + `\grabber2.exe`,
		ServiceExecutableExists: true,
		RequiredDefenderPaths:   []string{root},
		MissingDefenderPaths:    missingDefender,
		Services: []detector.ServiceState{{
			Name:   "ngs",
			Exists: true,
			Status: "running",
		}},
		Defender: detector.DefenderState{
			Available:      true,
			RequiredPaths:  []string{root},
			ExclusionPaths: exclusionsFor(root, missingDefender),
			MissingPaths:   missingDefender,
		},
	}
}

func brokenReport() detector.DetectionReport {
	report := healthyReport(nil)
	report.Health = detector.GrabberHealthBroken
	report.ServiceExecutableExists = false
	report.Services[0].Status = "stopped"
	return report
}

func exclusionsFor(root string, missing []string) []string {
	if len(missing) > 0 {
		return nil
	}
	return []string{root}
}
