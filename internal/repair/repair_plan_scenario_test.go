package repair_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kigrepair/internal/app"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
	"kigrepair/internal/logging"
	"kigrepair/internal/repair"
	"kigrepair/internal/reports"
	"kigrepair/internal/rollback"
	"kigrepair/internal/testfixtures"
)

func TestRepairDryRunScenarioReadiness(t *testing.T) {
	tests := []struct {
		name   string
		invite string
	}{
		{name: "invalid_installer_blocks_repair", invite: testfixtures.SecretInvite},
		{name: "missing_invite_blocks_repair_readiness"},
		{name: "missing_admin_blocks_repair_readiness", invite: testfixtures.SecretInvite},
		{name: "defender_deleted_service_binary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := testfixtures.ScenarioByName(tt.name)
			ctx := scenarioContext(t)
			recorder := &testfixtures.MutationRecorder{}
			plan, _, err := repair.DryRunWorkflow{
				Invite:  tt.invite,
				IsAdmin: func() bool { return scenario.IsAdmin },
				Detect:  scenario.DetectionReport,
				BuildCleanupPlan: func(report detector.DetectionReport) cleaner.CleanupPlan {
					return cleaner.BuildPlan(report, cleaner.PlanOptions{
						DryRun:       true,
						CleanupPaths: scenario.CleanupPaths(),
						PathExists:   scenario.PathExists,
						ValidatePath: func(string) error { return nil },
					})
				},
				InstallerResolver: scenario.InstallerResolver,
				PowerShellCheck:   available,
				MSIExecCheck:      available,
			}.Build(ctx)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if scenario.Expected.RepairPlanStatus != "" && plan.Status != scenario.Expected.RepairPlanStatus {
				t.Fatalf("repair plan status = %s, want %s; errors=%v", plan.Status, scenario.Expected.RepairPlanStatus, plan.Errors)
			}
			if scenario.Expected.ShouldRequireInvite && !containsString(plan.RequiredInputs, "invite") {
				t.Fatalf("required inputs = %#v, want invite", plan.RequiredInputs)
			}
			if scenario.Expected.ShouldRequireAdmin && !strings.Contains(strings.Join(plan.Errors, "\n"), "admin_rights") {
				t.Fatalf("errors = %#v, want admin_rights failure", plan.Errors)
			}
			testfixtures.AssertNoMutations(t, recorder)
		})
	}
}

func TestRepairDryRunScenarioWritesExpectedArtifactsAndNoSecret(t *testing.T) {
	scenario := testfixtures.ScenarioByName("defender_deleted_service_binary")
	scenario.IsAdmin = true
	scenario.Installer = testfixtures.ValidInstaller(filepath.Join(t.TempDir(), "grabberEM.x64.msi"))
	ctx := scenarioContext(t)
	err := repair.DryRunWorkflow{
		Invite:  testfixtures.SecretInvite,
		IsAdmin: func() bool { return true },
		Detect:  scenario.DetectionReport,
		BuildCleanupPlan: func(report detector.DetectionReport) cleaner.CleanupPlan {
			return cleaner.BuildPlan(report, cleaner.PlanOptions{
				DryRun:       true,
				CleanupPaths: scenario.CleanupPaths(),
				PathExists:   scenario.PathExists,
				ValidatePath: func(string) error { return nil },
			})
		},
		InstallerResolver: scenario.InstallerResolver,
		PowerShellCheck:   available,
		MSIExecCheck:      available,
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, name := range []string{
		"initial-detection.json",
		"classification-result.json",
		"recommendation-result.json",
		"preflight-result.json",
		"repair-plan.json",
		"cleanup-plan.json",
		"operations.json",
		"summary.txt",
	} {
		if _, err := os.Stat(filepath.Join(ctx.OutputDir, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
	testfixtures.AssertNoRawInvite(t, ctx.OutputDir, testfixtures.SecretInvite)
}

func TestRepairScenarioCreatesRollbackBeforeFirstMutation(t *testing.T) {
	scenario := testfixtures.ScenarioByName("defender_deleted_service_binary")
	scenario.IsAdmin = true
	scenario.Installer = testfixtures.ValidInstaller(filepath.Join(t.TempDir(), "grabberEM.x64.msi"))
	ctx := scenarioContext(t)
	recorder := &testfixtures.MutationRecorder{}
	order := []string{}

	err := repair.RepairWorkflow{
		Invite:    testfixtures.SecretInvite,
		Installer: scenario.Installer.Path,
		Yes:       true,
		IsAdmin:   func() bool { return true },
		Detect: sequenceReports(
			scenario.DetectionReport(),
			testfixtures.ScenarioByName("healthy_standard_install").DetectionReport(),
			testfixtures.ScenarioByName("healthy_standard_install").DetectionReport(),
		),
		BuildCleanupPlan: func(detector.DetectionReport) cleaner.CleanupPlan {
			return cleaner.CleanupPlan{Actions: []cleaner.CleanupAction{{
				Type:   cleaner.CleanupActionDeleteService,
				Target: "ngs",
				Safe:   true,
			}}}
		},
		InstallerResolver: scenario.InstallerResolver,
		CleanupExecutor: cleanupExecutorFunc(func(plan cleaner.CleanupPlan) []app.OperationResult {
			order = append(order, "mutation")
			return recorder.ExecutePlan(plan)
		}),
		MSIExecutor: fakeScenarioMSIExecutor{},
		CreateRollback: func(ctx *app.AppContext, initial detector.DetectionReport, plan cleaner.CleanupPlan, decision repair.Decision, installerPath string, hasInvite bool, isAdmin bool) (*rollback.RollbackInfo, error) {
			order = append(order, "rollback")
			return &rollback.RollbackInfo{Workflow: rollback.WorkflowRepair}, nil
		},
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(order) < 2 || order[0] != "rollback" || order[1] != "mutation" {
		t.Fatalf("order = %#v, want rollback before mutation", order)
	}
	testfixtures.AssertOnlyPlannedMutations(t, cleaner.CleanupPlan{Actions: []cleaner.CleanupAction{{Type: cleaner.CleanupActionDeleteService, Target: "ngs"}}}, recorder)
	testfixtures.AssertNoRawInvite(t, ctx.OutputDir, testfixtures.SecretInvite)
}

func TestRepairScenarioRollbackFailureAbortsBeforeMutation(t *testing.T) {
	scenario := testfixtures.ScenarioByName("defender_deleted_service_binary")
	scenario.IsAdmin = true
	scenario.Installer = testfixtures.ValidInstaller(filepath.Join(t.TempDir(), "grabberEM.x64.msi"))
	ctx := scenarioContext(t)
	recorder := &testfixtures.MutationRecorder{}

	err := repair.RepairWorkflow{
		Invite:    testfixtures.SecretInvite,
		Installer: scenario.Installer.Path,
		Yes:       true,
		IsAdmin:   func() bool { return true },
		Detect:    scenario.DetectionReport,
		BuildCleanupPlan: func(detector.DetectionReport) cleaner.CleanupPlan {
			return cleaner.CleanupPlan{Actions: []cleaner.CleanupAction{{Type: cleaner.CleanupActionDeleteService, Target: "ngs", Safe: true}}}
		},
		InstallerResolver: scenario.InstallerResolver,
		CleanupExecutor:   recorder,
		MSIExecutor:       fakeScenarioMSIExecutor{},
		CreateRollback: func(ctx *app.AppContext, initial detector.DetectionReport, plan cleaner.CleanupPlan, decision repair.Decision, installerPath string, hasInvite bool, isAdmin bool) (*rollback.RollbackInfo, error) {
			return nil, errRollbackFixture
		},
	}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if ctx.ExitCode != repair.ExitUnexpectedError {
		t.Fatalf("exit code = %d, want %d", ctx.ExitCode, repair.ExitUnexpectedError)
	}
	testfixtures.AssertNoMutations(t, recorder)
}

func available() (bool, error) { return true, nil }

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func scenarioContext(t *testing.T) *app.AppContext {
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

type cleanupExecutorFunc func(cleaner.CleanupPlan) []app.OperationResult

func (f cleanupExecutorFunc) ExecutePlan(plan cleaner.CleanupPlan) []app.OperationResult {
	return f(plan)
}

type fakeScenarioMSIExecutor struct{}

func (fakeScenarioMSIExecutor) Install(installerPath string, invite string, logPath string) installer.CommandResult {
	return installer.CommandResult{ExitCode: 0}
}

var errRollbackFixture = errors.New("fixture rollback write failed")

func sequenceReports(reports ...detector.DetectionReport) func() detector.DetectionReport {
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
