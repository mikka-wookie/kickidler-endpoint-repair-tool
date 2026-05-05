package wizard

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/classifier"
	"kigrepair/internal/detector"
	"kigrepair/internal/diagnostics"
	"kigrepair/internal/logging"
	"kigrepair/internal/recommendations"
	"kigrepair/internal/repair"
	"kigrepair/internal/verifier"
)

type Workflow struct {
	Options  Options
	Prompter Prompter
	In       io.Reader
	Out      io.Writer

	Detect        func() detector.DetectionReport
	RunRepair     func(*app.AppContext, Options) error
	RunCollect    func(*app.AppContext) error
	BuildDryRun   func(*app.AppContext, Options) (repair.RepairPlan, error)
	RunVerify     func(*app.AppContext) (*verifier.VerificationResult, error)
	ConfirmRepair func(string) bool
}

func (w Workflow) Name() string {
	return "wizard"
}

func (w Workflow) Run(ctx *app.AppContext) error {
	opts := w.Options
	opts.JSON = ctx.JSONOutput
	opts.Quiet = ctx.Quiet
	opts.NonInteractive = ctx.NonInteractive
	if strings.TrimSpace(opts.InviteValue) != "" {
		opts.HasInvite = true
	}
	result := Result{Command: "wizard", Status: StatusCompleted, ReportDir: ctx.OutputDir}
	ctx.JSONValue = result
	if opts.VerifyOnly && opts.AllowRepair {
		result.Status = StatusFailed
		result.Errors = append(result.Errors, "--verify-only cannot be combined with --repair")
		return w.finish(ctx, result)
	}

	addOperation(ctx, "wizard.started", "wizard", app.OperationStatusSuccess, "Wizard started", "")
	w.print(ctx, "kigrepair support wizard\n\nReport directory:\n%s\n\n", ctx.OutputDir)

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	detection := detect()
	result.DetectionHealth = string(detection.Health)
	result.InstallMode = string(detection.InstallMode)
	result.InstallRoot = detection.InstallRoot
	result.PrimaryService = detection.PrimaryService
	if err := ctx.Reporter.WriteJSON("initial-detection", detection); err != nil {
		return err
	}
	result.addStep("detection", stepStatusForHealth(detection.Health), "health: "+string(detection.Health), "initial-detection.json")
	addOperation(ctx, "wizard.detection", "grabber", operationStatus(result.Steps[len(result.Steps)-1].Status), "Detection completed", "")
	w.print(ctx, "Step 1/6: Checking current Grabber state...\nHealth: %s\nInstall mode: %s\n", detection.Health, detection.InstallMode)

	classification := classifier.Classify(classifier.ClassificationInput{Detection: &detection})
	if classification.PrimaryIssue != nil {
		result.ClassificationCode = classification.PrimaryIssue.Code
	}
	if err := ctx.Reporter.WriteJSON("classification-result", classification); err != nil {
		return err
	}
	result.addStep("classification", "success", classification.SupportSummary, "classification-result.json")
	addOperation(ctx, "wizard.classification", "grabber", app.OperationStatusSuccess, "Issue classification completed", "")

	recommendationClassification := recommendations.FromClassifier(classification)
	recommendation := recommendations.Plan(recommendations.RecommendationInput{
		Detection:      &detection,
		Classification: &recommendationClassification,
		HasInstaller:   strings.TrimSpace(opts.InstallerPath) != "",
		InstallerPath:  opts.InstallerPath,
		HasInvite:      opts.HasInvite,
		IsAdmin:        detection.IsAdmin,
		IsInteractive:  !ctx.NonInteractive && !ctx.Quiet,
		OutputDir:      ctx.OutputDir,
	})
	if recommendation.PrimaryAction != nil {
		result.RecommendationCode = recommendation.PrimaryAction.Code
	}
	if err := ctx.Reporter.WriteJSON("recommendation-result", recommendation); err != nil {
		return err
	}
	result.addStep("recommendation", "success", result.RecommendationCode, "recommendation-result.json")
	addOperation(ctx, "wizard.recommendation", "grabber", app.OperationStatusSuccess, "Recommendation generated", "")
	w.print(ctx, "Primary issue: %s\nRecommendation: %s\n\n", valueOrDash(result.ClassificationCode), valueOrDash(result.RecommendationCode))

	repairRecommended := recommendationRequiresRepair(recommendation)
	if opts.VerifyOnly || !repairRecommended {
		if err := w.verifyAndMaybeCollect(ctx, opts, &result, false); err != nil {
			return err
		}
		return w.finish(ctx, result)
	}

	opts = w.collectInputs(ctx, opts, &result)
	plan, err := w.buildDryRun(ctx, opts)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.addStep("repair dry-run", "failed", err.Error())
		addOperation(ctx, "wizard.repair_plan", "repair", app.OperationStatusFailed, "Repair dry-run failed", err.Error())
		return w.finish(ctx, result)
	}
	result.PreflightStatus = preflightStatus(plan)
	result.RepairPlanStatus = plan.Status
	result.Warnings = append(result.Warnings, plan.Warnings...)
	result.Errors = append(result.Errors, plan.Errors...)
	result.addStep("preflight", preflightStepStatus(plan), result.PreflightStatus, "preflight-result.json")
	result.addStep("repair dry-run", repairPlanStepStatus(plan), plan.Status, "repair-plan.json", "cleanup-plan.json")
	addOperation(ctx, "wizard.preflight", "repair", operationStatus(preflightStepStatus(plan)), "Preflight completed", strings.Join(plan.Errors, "; "))
	addOperation(ctx, "wizard.repair_plan", "repair", operationStatus(repairPlanStepStatus(plan)), "Repair dry-run completed", strings.Join(plan.Errors, "; "))
	w.print(ctx, "Preflight: %s\nRepair dry-run: %s\n", result.PreflightStatus, result.RepairPlanStatus)

	ready := plan.ReadyForRepair && plan.Status != repair.RepairPlanStatusNotReady && plan.Status != repair.RepairPlanStatusFailed
	if !ready {
		result.Status = StatusNotReady
		result.addStep("repair", "skipped", "repair is not ready")
		if err := w.verifyAndMaybeCollect(ctx, opts, &result, true); err != nil {
			return err
		}
		return w.finish(ctx, result)
	}
	if !opts.AllowRepair {
		result.Status = StatusCompletedWithWarnings
		result.Warnings = append(result.Warnings, "repair is ready but --repair was not provided")
		result.addStep("repair", "skipped", "repair requires --repair and explicit confirmation")
		if err := w.verifyAndMaybeCollect(ctx, opts, &result, true); err != nil {
			return err
		}
		return w.finish(ctx, result)
	}
	if !w.repairConfirmed(ctx, opts) {
		result.Status = StatusCancelled
		result.addStep("repair", "skipped", "confirmation declined")
		addOperation(ctx, "wizard.repair_confirmation", "user", app.OperationStatusSkipped, "Repair confirmation declined", "")
		if err := w.verifyAndMaybeCollect(ctx, opts, &result, true); err != nil {
			return err
		}
		return w.finish(ctx, result)
	}

	err = w.runRepair(ctx, opts)
	result.RepairExecuted = true
	if err != nil || ctx.ExitCode != app.ExitSuccess {
		result.Status = StatusRepairFailed
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
		result.addStep("repair", "failed", "real repair failed", "repair-result.json")
	} else {
		result.Status = StatusRepairCompleted
		result.addStep("repair", "success", "real repair completed", "repair-result.json")
	}
	if err := w.verifyAndMaybeCollect(ctx, opts, &result, result.Status == StatusRepairFailed); err != nil {
		return err
	}
	return w.finish(ctx, result)
}

func (w Workflow) collectInputs(ctx *app.AppContext, opts Options, result *Result) Options {
	if ctx.NonInteractive {
		return opts
	}
	prompter := w.prompter()
	if strings.TrimSpace(opts.InstallerPath) == "" {
		if value, err := prompter.AskString("Enter path to Grabber MSI installer, or press Enter to skip repair", true); err == nil && strings.TrimSpace(value) != "" {
			opts.InstallerPath = strings.TrimSpace(value)
			result.addStep("installer input", "success", "installer path provided")
		} else {
			result.addStep("installer input", "skipped", "installer path not provided")
		}
	}
	if !opts.HasInvite {
		if value, err := prompter.AskSecret("Enter Kickidler invite, or press Enter to skip repair", true); err == nil && strings.TrimSpace(value) != "" {
			opts.InviteValue = strings.TrimSpace(value)
			opts.HasInvite = true
			result.addStep("invite input", "success", "invite provided")
		} else {
			result.addStep("invite input", "skipped", "invite not provided")
		}
	}
	return opts
}

func (w Workflow) buildDryRun(ctx *app.AppContext, opts Options) (repair.RepairPlan, error) {
	if w.BuildDryRun != nil {
		return w.BuildDryRun(ctx, opts)
	}
	before := ctx.ExitCode
	err := repair.DryRunWorkflow{Invite: opts.InviteValue, Installer: opts.InstallerPath, Yes: opts.Yes}.Run(ctx)
	if value, ok := ctx.JSONValue.(repair.RepairPlanJSON); ok {
		ctx.ExitCode = before
		return value.RepairPlan, err
	}
	ctx.ExitCode = before
	return repair.RepairPlan{Status: repair.RepairPlanStatusFailed, ReadyForRepair: false}, err
}

func (w Workflow) repairConfirmed(ctx *app.AppContext, opts Options) bool {
	if ctx.NonInteractive {
		return opts.AllowRepair && opts.Yes
	}
	if w.ConfirmRepair != nil {
		return w.ConfirmRepair("Run real repair now?")
	}
	prompter := w.prompter()
	answer, err := prompter.AskString("Real repair is ready.\n\nThis will modify the system:\n- stop/delete Grabber services\n- terminate trusted Grabber processes\n- delete validated leftovers\n- add Defender exclusion if missing\n- run MSI install\n\nType YES to run real repair", true)
	return err == nil && answer == "YES"
}

func (w Workflow) runRepair(ctx *app.AppContext, opts Options) error {
	if w.RunRepair != nil {
		return w.RunRepair(ctx, opts)
	}
	return repair.RepairWorkflow{Invite: opts.InviteValue, Installer: opts.InstallerPath, Yes: true}.Run(ctx)
}

func (w Workflow) verifyAndMaybeCollect(ctx *app.AppContext, opts Options, result *Result, forceCollect bool) error {
	verification, err := w.runVerify(ctx)
	if err != nil {
		result.addStep("verify", "failed", err.Error(), "verification-result.json")
		result.Errors = append(result.Errors, err.Error())
	} else if verification != nil {
		result.VerifyStatus = string(verification.OverallStatus)
		result.addStep("verify", verifyStepStatus(*verification), result.VerifyStatus, "verification-result.json")
	}
	collect := opts.CollectBundle || forceCollect || strings.EqualFold(result.ClassificationCode, classifier.CodeUnknownInstallState)
	if !collect && !ctx.NonInteractive && !ctx.Quiet {
		confirmed, promptErr := w.prompter().Confirm("Collect support bundle now?", false)
		if promptErr != nil {
			result.Warnings = append(result.Warnings, promptErr.Error())
		}
		collect = confirmed
	}
	if collect {
		_ = ctx.Reporter.WriteJSON("wizard-result", *result)
		if err := w.runCollect(ctx); err != nil {
			result.addStep("collect-report", "failed", err.Error(), "collect-result.json")
			result.Warnings = append(result.Warnings, err.Error())
		} else {
			result.BundlePath = ctx.OutputDir + string(os.PathSeparator) + "kigrepair-support-bundle.zip"
			result.addStep("collect-report", "success", "support bundle collected", "collect-result.json", "kigrepair-support-bundle.zip")
		}
	} else {
		result.addStep("collect-report", "skipped", "not requested")
	}
	return nil
}

func (w Workflow) runVerify(ctx *app.AppContext) (*verifier.VerificationResult, error) {
	if w.RunVerify != nil {
		return w.RunVerify(ctx)
	}
	before := ctx.ExitCode
	err := verifier.VerifyWorkflow{}.Run(ctx)
	if value, ok := ctx.JSONValue.(verifier.VerifyResult); ok {
		ctx.ExitCode = before
		return &value.VerificationResult, err
	}
	ctx.ExitCode = before
	return nil, err
}

func (w Workflow) runCollect(ctx *app.AppContext) error {
	if w.RunCollect != nil {
		return w.RunCollect(ctx)
	}
	before := ctx.ExitCode
	err := diagnostics.CollectReportWorkflow{IncludeEventLogs: true, IncludeHistory: true, HistoryLimit: 5}.Run(ctx)
	ctx.ExitCode = before
	return err
}

func (w Workflow) finish(ctx *app.AppContext, result Result) error {
	if result.Status == "" {
		result.Status = StatusCompleted
	}
	if result.Status == StatusCompleted && hasWarnings(result) {
		result.Status = StatusCompletedWithWarnings
	}
	result.ExitCode = ExitCode(result.Status)
	ctx.ExitCode = result.ExitCode
	ctx.JSONValue = result
	if err := ctx.Reporter.WriteJSON("wizard-result", result); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteText("summary", FormatSummary(result)); err != nil {
		return err
	}
	addOperation(ctx, "wizard.completed", "wizard", operationStatusForExit(result.ExitCode), "Wizard completed with status: "+result.Status, strings.Join(result.Errors, "; "))
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatSummary(result))
	}
	return nil
}

func (w Workflow) print(ctx *app.AppContext, format string, args ...any) {
	if ctx.Quiet || ctx.JSONOutput {
		return
	}
	out := w.Out
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintf(out, format, args...)
}

func (w Workflow) prompter() Prompter {
	if w.Prompter != nil {
		return w.Prompter
	}
	in := w.In
	if in == nil {
		in = os.Stdin
	}
	out := w.Out
	if out == nil {
		out = os.Stdout
	}
	return ConsolePrompter{In: in, Out: out}
}

func (r *Result) addStep(name string, status string, summary string, artifacts ...string) {
	r.Steps = append(r.Steps, StepResult{Name: name, Status: status, Summary: summary, Artifacts: artifacts})
}

func addOperation(ctx *app.AppContext, step string, target string, status app.OperationStatus, message string, err string) {
	operation := app.OperationResult{Step: step, Target: target, Status: status, Message: message, Error: err, Timestamp: time.Now()}
	ctx.AddResult(operation)
	if ctx.Logger != nil {
		logging.LogOperation(ctx.Logger, operation)
	}
}

func recommendationRequiresRepair(result recommendations.RecommendationResult) bool {
	if result.PrimaryAction == nil {
		return false
	}
	switch result.PrimaryAction.Code {
	case recommendations.ActionNoRepairRequired, recommendations.ActionCollectSupportBundle, recommendations.ActionEscalateWithBundle:
		return false
	default:
		return result.PrimaryAction.Destructive || contains(result.PrimaryAction.RequiredInputs, recommendations.InputInstaller) || contains(result.PrimaryAction.RequiredInputs, recommendations.InputInvite)
	}
}

func preflightStatus(plan repair.RepairPlan) string {
	if plan.Preflight == nil {
		return ""
	}
	if plan.Preflight.HasFailedRequiredCheck() {
		return "not_ready"
	}
	if len(plan.Preflight.Warnings) > 0 {
		return "ready_with_warnings"
	}
	return "ready"
}

func preflightStepStatus(plan repair.RepairPlan) string {
	switch preflightStatus(plan) {
	case "not_ready":
		return "warning"
	case "ready_with_warnings":
		return "warning"
	case "ready":
		return "success"
	default:
		return "skipped"
	}
}

func repairPlanStepStatus(plan repair.RepairPlan) string {
	switch plan.Status {
	case repair.RepairPlanStatusPlanned:
		return "success"
	case repair.RepairPlanStatusPlannedWithWarnings, repair.RepairPlanStatusNotReady:
		return "warning"
	default:
		return "failed"
	}
}

func stepStatusForHealth(health detector.GrabberHealthStatus) string {
	if health == detector.GrabberHealthHealthy {
		return "success"
	}
	if health == detector.GrabberHealthUnknown {
		return "failed"
	}
	return "warning"
}

func verifyStepStatus(result verifier.VerificationResult) string {
	switch result.OverallStatus {
	case verifier.VerificationPassed:
		return "success"
	case verifier.VerificationWarning, verifier.VerificationSkipped:
		return "warning"
	default:
		return "failed"
	}
}

func operationStatus(status string) app.OperationStatus {
	switch status {
	case "success":
		return app.OperationStatusSuccess
	case "failed":
		return app.OperationStatusFailed
	case "skipped":
		return app.OperationStatusSkipped
	default:
		return app.OperationStatusWarning
	}
}

func operationStatusForExit(exitCode int) app.OperationStatus {
	if exitCode == app.ExitSuccess {
		return app.OperationStatusSuccess
	}
	if exitCode == app.ExitUnexpectedError {
		return app.OperationStatusFailed
	}
	return app.OperationStatusWarning
}

func hasWarnings(result Result) bool {
	if len(result.Warnings) > 0 {
		return true
	}
	for _, step := range result.Steps {
		if step.Status == "warning" {
			return true
		}
	}
	return false
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
