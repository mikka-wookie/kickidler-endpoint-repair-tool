package repair

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/classifier"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/defender"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
	"kigrepair/internal/logging"
	"kigrepair/internal/recommendations"
	"kigrepair/internal/rollback"
	"kigrepair/internal/verifier"
	"kigrepair/internal/version"
)

type CleanupExecutor interface {
	ExecutePlan(cleaner.CleanupPlan) []app.OperationResult
}

type RepairWorkflow struct {
	Invite    string `json:"-"`
	Installer string `json:"installer,omitempty"`
	Yes       bool   `json:"yes,omitempty"`

	Detect            func() detector.DetectionReport
	IsAdmin           func() bool
	BuildCleanupPlan  func(detector.DetectionReport) cleaner.CleanupPlan
	CleanupExecutor   CleanupExecutor
	MSIExecutor       installer.MSIExecutor
	InstallerResolver func(string) (installer.InstallerResolution, error)
	DefenderAdder     defender.ExclusionAdder
	CreateRollback    func(ctx *app.AppContext, initial detector.DetectionReport, plan cleaner.CleanupPlan, decision Decision, installerPath string, hasInvite bool, isAdmin bool) (*rollback.RollbackInfo, error)
	In                io.Reader
	Out               io.Writer
}

func (w RepairWorkflow) Name() string {
	return "repair"
}

func (w RepairWorkflow) Run(ctx *app.AppContext) error {
	startedAt := time.Now()
	result := RepairResult{
		Build:          version.Get(),
		StartedAt:      startedAt,
		Mode:           string(ctx.Mode),
		Force:          ctx.Force,
		InviteProvided: strings.TrimSpace(w.Invite) != "",
		ReportDir:      ctx.OutputDir,
	}
	ctx.JSONValue = result
	ctx.Logger.Info("repair started")
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)
	ctx.Logger.Info("invite provided: %t", result.InviteProvided)
	ctx.Logger.Info("force: %t", ctx.Force)
	addOperation(ctx, "repair_started", "grabber", app.OperationStatusSuccess, "Repair workflow started", "")

	invite, ok := w.validateInputs(ctx, &result)
	if !ok {
		return w.finish(ctx, result, nil)
	}
	addOperation(ctx, "validate_inputs", "repair-inputs", app.OperationStatusSuccess, "Repair inputs are valid", "")

	isAdmin := checks.IsAdmin
	if w.IsAdmin != nil {
		isAdmin = w.IsAdmin
	}
	admin := isAdmin()
	ctx.Logger.Info("admin status: %t", admin)
	if !admin {
		message := "Administrator rights are required for repair"
		ctx.ExitCode = ExitAdminRequired
		result.Errors = append(result.Errors, message)
		addOperation(ctx, "check_admin", "administrator", app.OperationStatusFailed, message, message)
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Fprintln(os.Stderr, message)
		}
		return w.finish(ctx, result, nil)
	}
	addOperation(ctx, "check_admin", "administrator", app.OperationStatusSuccess, "Administrator rights confirmed", "")

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	initial := detect()
	result.InitialHealth = string(initial.Health)
	result.InitialInstallMode = string(initial.InstallMode)
	result.InitialInstallRoot = initial.InstallRoot
	ctx.Logger.Info("initial health/mode/root: %s / %s / %s", initial.Health, initial.InstallMode, initial.InstallRoot)
	if err := ctx.Reporter.WriteJSON("initial-detection", initial); err != nil {
		return err
	}
	addOperation(ctx, "initial_detection", "grabber", detectionStatus(initial), "Initial detection completed with health: "+string(initial.Health), "")

	plan := w.buildCleanupPlan(initial)
	plan.Actions = cleaner.SortActionsForExecution(plan.Actions)
	if err := ctx.Reporter.WriteJSON("cleanup-plan", plan); err != nil {
		return err
	}
	addPlanOperations(ctx, plan)

	decision := Decide(initial, ctx.Force, len(plan.Actions))
	result.CleanupNeeded = decision.CleanupNeeded
	result.InstallExecuted = false
	result.DefenderExecuted = false
	result.Warnings = append(result.Warnings, decision.Warnings...)
	result.Errors = append(result.Errors, decision.Errors...)
	ctx.Logger.Info("repair decision: cleanup=%t install=%t defender=%t stop=%t", decision.CleanupNeeded, decision.InstallNeeded, decision.DefenderNeeded, decision.Stop)
	addOperation(ctx, "repair_decision", "grabber", decisionStatus(decision), decisionMessage(decision), strings.Join(decision.Errors, "; "))

	if decision.Stop {
		ctx.ExitCode = ExitWarnings
		return w.finish(ctx, result, &initial)
	}

	var resolution installer.InstallerResolution
	var installerPath string
	if decision.InstallNeeded {
		resolved, validation, err := w.resolveAndValidateInstaller(ctx)
		resolution = resolved
		result.InstallerResolution = &resolution
		result.InstallerValidation = validation
		if validation != nil {
			if err := ctx.Reporter.WriteJSON("installer-validation", validation); err != nil {
				return err
			}
			addInstallerValidationOperation(ctx, validation.Path, *validation)
			result.Warnings = append(result.Warnings, validation.Warnings...)
		}
		if err != nil {
			ctx.ExitCode = ExitInvalidInput
			result.Errors = append(result.Errors, err.Error())
			addOperation(ctx, "resolve_installer", "installer", app.OperationStatusFailed, "Installer could not be resolved", err.Error())
			return w.finish(ctx, result, &initial)
		}
		if validation != nil && !validation.IsUsable() {
			message := "Installer cannot be used: " + validation.ErrorSummary()
			ctx.ExitCode = ExitInvalidInput
			result.Errors = append(result.Errors, message)
			addOperation(ctx, "validate_installer", validation.Path, app.OperationStatusFailed, message, strings.Join(validation.Errors, "; "))
			return w.finish(ctx, result, &initial)
		}
		installerPath = resolution.SelectedPath
		result.InstallerPath = installerPath
		addOperation(ctx, "resolve_installer", installerPath, app.OperationStatusSuccess, "Installer resolved from "+string(resolution.SelectedSource), "")
	}

	if err := w.confirm(ctx, initial, decision); err != nil {
		ctx.ExitCode = ExitConfirmationRequired
		message := err.Error()
		if errors.Is(err, ErrRepairConfirmationRequired) {
			message = "Repair in non-interactive mode requires --yes"
		}
		result.Errors = append(result.Errors, message)
		addOperation(ctx, "repair_confirmation", "user", app.OperationStatusFailed, message, message)
		ctx.Logger.Warn("confirmation status: declined or missing")
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Fprintln(os.Stderr, message)
		}
		return w.finish(ctx, result, &initial)
	}
	addOperation(ctx, "repair_confirmation", "user", app.OperationStatusSuccess, "Repair confirmed", "")

	if decision.CleanupNeeded && len(plan.Blockers) > 0 {
		ctx.ExitCode = ExitCleanupFailed
		result.Errors = append(result.Errors, plan.Blockers...)
		addOperation(ctx, "cleanup_execute", "cleanup-plan", app.OperationStatusFailed, "Cleanup blocked by safety validation", strings.Join(plan.Blockers, "; "))
		return w.finish(ctx, result, &initial)
	}

	rollbackInfo, err := w.createRollback(ctx, initial, plan, decision, installerPath, result.InviteProvided, admin)
	if err != nil {
		ctx.ExitCode = ExitUnexpectedError
		message := "Rollback snapshot failed"
		result.Errors = append(result.Errors, message+": "+err.Error())
		addOperation(ctx, "rollback.snapshot_capture", rollback.Path(ctx.OutputDir), app.OperationStatusFailed, message, err.Error())
		ctx.Logger.Error("rollback snapshot failed: %s", err)
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Fprintln(os.Stderr, "Rollback snapshot: failed")
			fmt.Fprintln(os.Stderr, "No system changes were made.")
			fmt.Fprintln(os.Stderr, "Reason: could not write rollback-info.json")
		}
		return w.finish(ctx, result, &initial)
	}
	ledger := rollback.NewLedger(rollbackInfo)
	result.Rollback = ptrRollbackSummary(rollback.SummaryFromInfo(rollback.Path(ctx.OutputDir), rollbackInfo))
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Println("Rollback snapshot: created")
		fmt.Println("Snapshot file: " + rollback.Path(ctx.OutputDir))
		fmt.Println("Restore supported: no")
	}
	executionStart := len(ctx.Results)

	if decision.CleanupNeeded {
		result.CleanupExecuted = true
		executor := w.CleanupExecutor
		if executor == nil {
			configured := cleaner.NewExecutor(ctx.OutputDir, ctx.Logger)
			if ctx.Config.Repair.ServiceStopTimeoutSeconds > 0 {
				configured.ServiceTimeout = time.Duration(ctx.Config.Repair.ServiceStopTimeoutSeconds) * time.Second
			}
			if ctx.Config.Repair.ProcessKillTimeoutSeconds > 0 {
				configured.ProcessTimeout = time.Duration(ctx.Config.Repair.ProcessKillTimeoutSeconds) * time.Second
			}
			executor = configured
		}
		for _, operation := range executor.ExecutePlan(plan) {
			ctx.AddResult(operation)
			classifyOperation(&result, operation)
		}
		for _, change := range cleaner.RollbackExecutedChanges(ctx.Results[executionStart:], rollbackInfo.PlannedChanges) {
			rollback.RecordExecutedChange(ledger, change)
		}
		executionStart = len(ctx.Results)
		if err := ctx.Reporter.WriteJSON("cleanup-result", cleanupResult(plan, ctx.Results)); err != nil {
			return err
		}
	}

	if decision.CleanupNeeded {
		status := app.OperationStatusSuccess
		message := "Cleanup completed"
		if cleanupFailed(ctx.Results) {
			status = app.OperationStatusWarning
			message = "Cleanup completed with failures; continuing repair"
			result.Warnings = append(result.Warnings, message)
		}
		addOperation(ctx, "cleanup_execute", "cleanup-plan", status, message, "")
	} else {
		addOperation(ctx, "cleanup_execute", "cleanup-plan", app.OperationStatusSkipped, "Cleanup was not required", "")
	}

	var msi installer.MSIResult
	if decision.InstallNeeded {
		result.InstallExecuted = true
		msi = w.runInstall(ctx, installerPath, invite)
		for _, operation := range ctx.Results[executionStart:] {
			if operation.Step == "msi_install" {
				rollback.RecordExecutedChange(ledger, rollback.ExecutedChangeFromOperation(operation, rollbackInfo.PlannedChanges))
			}
		}
		executionStart = len(ctx.Results)
		result.RebootRequired = msi.RebootRequired
		result.Rollback = ptrRollbackSummary(rollback.SummaryFromInfo(rollback.Path(ctx.OutputDir), rollbackInfo))
		if err := rollback.WriteFile(ctx.OutputDir, rollbackInfo); err != nil {
			return err
		}
		addOperation(ctx, "rollback.info_written", rollback.Path(ctx.OutputDir), app.OperationStatusSuccess, "Rollback info written", "")
		if err := ctx.Reporter.WriteJSON("install-result", installResult(startedAt, string(ctx.Mode), installerPath, result.InviteProvided, msi, ctx.OutputDir, resolution, result.InstallerValidation)); err != nil {
			return err
		}
		if !msi.Success {
			ctx.ExitCode = ExitInstallFailed
			result.Errors = append(result.Errors, msi.Message)
			return w.finish(ctx, result, &initial)
		}
	} else {
		msi = installer.MSIResult{ExitCode: -1, Status: "not_run", Success: true, Message: "MSI install was not required"}
		addOperation(ctx, "msi_install", "installer", app.OperationStatusSkipped, "MSI install was not required", "")
	}

	defenderResult, defenderRan := w.runDefenderEnsure(ctx, detect, decision.DefenderNeeded)
	result.DefenderExecuted = defenderRan
	for _, operation := range ctx.Results[executionStart:] {
		if operation.Step == "defender_ensure" {
			rollback.RecordExecutedChange(ledger, rollback.ExecutedChangeFromOperation(operation, rollbackInfo.PlannedChanges))
		}
	}
	if defenderRan {
		if err := ctx.Reporter.WriteJSON("defender-result", defenderResult); err != nil {
			return err
		}
		result.Warnings = append(result.Warnings, defenderResult.Warnings...)
		result.Errors = append(result.Errors, defenderResult.Errors...)
	}

	final := detect()
	result.FinalHealth = string(final.Health)
	result.FinalInstallMode = string(final.InstallMode)
	result.FinalInstallRoot = final.InstallRoot
	ctx.Logger.Info("final health/mode/root: %s / %s / %s", final.Health, final.InstallMode, final.InstallRoot)
	if err := ctx.Reporter.WriteJSON("final-detection", final); err != nil {
		return err
	}
	addOperation(ctx, "final_detection", "grabber", detectionStatus(final), "Final detection completed with health: "+string(final.Health), "")

	verification := verifier.VerifyInstallation(final, verifier.VerifyOptions{
		InstallExecuted:            result.InstallExecuted,
		MSIInstallLogPath:          msiInstallLogPath(ctx.OutputDir),
		RequireRunningProcess:      true,
		AllowDefenderUnavailable:   true,
		ExpectInstalledState:       true,
		AllowStoppedServiceWarning: true,
	})
	result.Verification = &verification
	classification := classifier.Classify(classifier.ClassificationInput{Detection: &final, Verification: &verification})
	result.Classification = &classification
	result.Warnings = append(result.Warnings, verification.Warnings...)
	result.Errors = append(result.Errors, verification.Errors...)
	for _, operation := range verifier.Operations(verification) {
		ctx.AddResult(operation)
		logging.LogOperation(ctx.Logger, operation)
	}
	ctx.Logger.Info("final verification result: %s", verification.OverallStatus)

	ctx.ExitCode = finalExitCode(msi, verification, ctx.Results, defenderRan)
	result.Rollback = ptrRollbackSummary(rollback.SummaryFromInfo(rollback.Path(ctx.OutputDir), rollbackInfo))
	addOperation(ctx, "rollback.executed_changes_recorded", rollback.Path(ctx.OutputDir), app.OperationStatusSuccess, "Rollback executed changes recorded", "")
	if err := rollback.WriteFile(ctx.OutputDir, rollbackInfo); err != nil {
		return err
	}
	addOperation(ctx, "rollback.info_written", rollback.Path(ctx.OutputDir), app.OperationStatusSuccess, "Rollback info written", "")
	return w.finish(ctx, result, &final)
}

func (w RepairWorkflow) resolveAndValidateInstaller(ctx *app.AppContext) (installer.InstallerResolution, *installer.ValidationResult, error) {
	resolveInstaller := installer.ResolveInstaller
	if w.InstallerResolver != nil {
		resolveInstaller = w.InstallerResolver
	}
	resolution, err := resolveInstaller(w.Installer)
	installer.LogInstallerResolution(ctx.Logger, resolution)
	validation := installer.ValidationFromResolution(resolution)
	if validation != nil {
		ctx.Logger.Info("installer validation status: %s", validation.Status)
	}
	if err != nil {
		return resolution, validation, err
	}
	return resolution, validation, nil
}

func (w RepairWorkflow) validateInputs(ctx *app.AppContext, result *RepairResult) (string, bool) {
	invite, err := installer.ValidateInvite(w.Invite)
	if err != nil {
		ctx.ExitCode = ExitInvalidInput
		result.Errors = append(result.Errors, err.Error())
		addOperation(ctx, "validate_inputs", "invite", app.OperationStatusFailed, "Invalid invite", err.Error())
		return "", false
	}
	return invite, true
}

func (w RepairWorkflow) buildCleanupPlan(report detector.DetectionReport) cleaner.CleanupPlan {
	if w.BuildCleanupPlan != nil {
		return w.BuildCleanupPlan(report)
	}
	return cleaner.BuildPlan(report, cleaner.PlanOptions{DryRun: false})
}

func (w RepairWorkflow) createRollback(ctx *app.AppContext, initial detector.DetectionReport, plan cleaner.CleanupPlan, decision Decision, installerPath string, hasInvite bool, isAdmin bool) (*rollback.RollbackInfo, error) {
	if w.CreateRollback != nil {
		return w.CreateRollback(ctx, initial, plan, decision, installerPath, hasInvite, isAdmin)
	}
	planned := cleaner.RollbackPlannedChanges(plan, rollback.WorkflowRepair)
	if decision.InstallNeeded {
		planned = append(planned, rollback.PlannedChange{
			ID:          fmt.Sprintf("%s-%03d", rollback.WorkflowRepair, len(planned)+1),
			Type:        rollback.TypeMSIInstall,
			Target:      installerPath,
			Destructive: true,
			Reason:      "Repair requires MSI installation",
			Source:      rollback.WorkflowRepair,
		})
	}
	if decision.DefenderNeeded {
		required, _ := defender.RequiredPaths(initial, false)
		for _, path := range required {
			if detector.IsPathCoveredByAnyExclusion(path, initial.Defender.ExclusionPaths) {
				continue
			}
			planned = append(planned, rollback.PlannedChange{
				ID:          fmt.Sprintf("%s-%03d", rollback.WorkflowRepair, len(planned)+1),
				Type:        rollback.TypeDefenderAddExclusion,
				Target:      path,
				Destructive: false,
				Reason:      "Repair requires Defender exclusion",
				Source:      rollback.WorkflowRepair,
			})
		}
	}
	info, err := rollback.CaptureSnapshot(context.Background(), rollback.SnapshotInput{
		ReportDir:      ctx.OutputDir,
		Workflow:       rollback.WorkflowRepair,
		Detection:      &initial,
		InstallerPath:  installerPath,
		HasInvite:      hasInvite,
		IsAdmin:        isAdmin,
		FileTargets:    cleaner.RollbackFileTargets(plan),
		RegistryKeys:   cleaner.RollbackRegistryTargets(plan),
		RequiredPaths:  initial.RequiredDefenderPaths,
		PlannedChanges: planned,
	})
	if err != nil {
		return nil, err
	}
	if err := rollback.WriteFile(ctx.OutputDir, info); err != nil {
		return nil, err
	}
	addOperation(ctx, "rollback.snapshot_capture", rollback.Path(ctx.OutputDir), app.OperationStatusSuccess, "Rollback snapshot captured", "")
	addOperation(ctx, "rollback.planned_changes_recorded", rollback.Path(ctx.OutputDir), app.OperationStatusSuccess, "Rollback planned changes recorded", "")
	addOperation(ctx, "rollback.info_written", rollback.Path(ctx.OutputDir), app.OperationStatusSuccess, "Rollback info written", "")
	ctx.Logger.Info("rollback snapshot created: %s", rollback.Path(ctx.OutputDir))
	return info, nil
}

func ptrRollbackSummary(summary rollback.Summary) *rollback.Summary {
	return &summary
}

func (w RepairWorkflow) confirm(ctx *app.AppContext, initial detector.DetectionReport, decision Decision) error {
	if !ctx.Quiet && !ctx.JSONOutput && !w.Yes {
		fmt.Print(FormatPreflight(initial, decision, ctx.OutputDir))
	}
	in := w.In
	if in == nil {
		in = os.Stdin
	}
	out := w.Out
	if out == nil {
		out = os.Stdout
		if ctx.JSONOutput {
			out = os.Stderr
		}
	}
	return ConfirmRepair(ctx, w.Yes, in, out)
}

func (w RepairWorkflow) finish(ctx *app.AppContext, result RepairResult, final *detector.DetectionReport) error {
	result.FinishedAt = time.Now()
	result.ExitCode = ctx.ExitCode
	var recommendationClassification *recommendations.ClassificationResult
	if result.Classification != nil {
		converted := recommendations.FromClassifier(*result.Classification)
		recommendationClassification = &converted
	}
	recommendation := recommendations.Plan(recommendations.RecommendationInput{
		Detection:      final,
		Verification:   recommendationVerification(result.Verification),
		Classification: recommendationClassification,
		InstallerPath:  result.InstallerPath,
		HasInstaller:   strings.TrimSpace(result.InstallerPath) != "" || result.InstallerResolution != nil,
		HasInvite:      result.InviteProvided,
		IsAdmin:        final == nil || final.IsAdmin,
		IsInteractive:  !ctx.NonInteractive && !ctx.Quiet,
		OutputDir:      ctx.OutputDir,
		RepairFailed:   repairHardFailed(ctx.ExitCode),
	})
	result.Recommendation = &recommendation
	ctx.JSONValue = result
	ctx.Logger.Info("cleanup needed: %t", result.CleanupNeeded)
	ctx.Logger.Info("cleanup executed: %t", result.CleanupExecuted)
	ctx.Logger.Info("install executed: %t", result.InstallExecuted)
	ctx.Logger.Info("defender executed: %t", result.DefenderExecuted)
	ctx.Logger.Info("final health/mode/root: %s / %s / %s", result.FinalHealth, result.FinalInstallMode, result.FinalInstallRoot)
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)
	status := app.OperationStatusSuccess
	if ctx.ExitCode != 0 {
		status = app.OperationStatusWarning
	}
	if ctx.ExitCode >= ExitInvalidInput && ctx.ExitCode != ExitRebootRequired {
		status = app.OperationStatusFailed
	}
	addOperation(ctx, "repair_completed", "grabber", status, fmt.Sprintf("Repair workflow completed with exit code %d", ctx.ExitCode), strings.Join(result.Errors, "; "))
	if err := ctx.Reporter.WriteJSON("repair-result", result); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteJSON("recommendation-result", recommendation); err != nil {
		return err
	}
	if result.Verification != nil {
		if err := ctx.Reporter.WriteJSON("verification-result", result.Verification); err != nil {
			return err
		}
	}
	if result.Classification != nil {
		if err := ctx.Reporter.WriteJSON("classification-result", result.Classification); err != nil {
			return err
		}
	}
	if result.InstallerValidation != nil {
		if err := ctx.Reporter.WriteJSON("installer-validation", result.InstallerValidation); err != nil {
			return err
		}
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	summary := FormatSummary(result, final)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func addInstallerValidationOperation(ctx *app.AppContext, target string, validation installer.ValidationResult) {
	status := app.OperationStatusSuccess
	if validation.Status == installer.ValidationStatusValidWithWarnings {
		status = app.OperationStatusWarning
	}
	if !validation.IsUsable() {
		status = app.OperationStatusFailed
	}
	addOperation(ctx, "installer_validation", target, status, "Installer validation: "+validation.Status, strings.Join(validation.Errors, "; "))
}

func repairHardFailed(exitCode int) bool {
	switch exitCode {
	case ExitCleanupFailed, ExitInstallFailed, ExitVerificationFailed, ExitDefenderFailed, ExitInvalidInput, ExitUnexpectedError:
		return true
	default:
		return false
	}
}

func recommendationVerification(result *verifier.VerificationResult) *recommendations.VerificationState {
	if result == nil {
		return nil
	}
	return &recommendations.VerificationState{
		OverallStatus: string(result.OverallStatus),
		Warnings:      append([]string{}, result.Warnings...),
		Errors:        append([]string{}, result.Errors...),
	}
}
