package repair

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/classifier"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/defender"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
	"kigrepair/internal/recommendations"
)

const (
	RepairPlanStatusPlanned             = "planned"
	RepairPlanStatusPlannedWithWarnings = "planned_with_warnings"
	RepairPlanStatusNotReady            = "not_ready"
	RepairPlanStatusFailed              = "failed"
)

type DryRunWorkflow struct {
	Invite    string `json:"-"`
	Installer string `json:"installer,omitempty"`
	Yes       bool   `json:"yes,omitempty"`

	Detect            func() detector.DetectionReport
	IsAdmin           func() bool
	BuildCleanupPlan  func(detector.DetectionReport) cleaner.CleanupPlan
	InstallerResolver func(string) (installer.InstallerResolution, error)
	PowerShellCheck   func() (bool, error)
	MSIExecCheck      func() (bool, error)
}

func (w DryRunWorkflow) Name() string {
	return "repair --dry-run"
}

func (w DryRunWorkflow) Run(ctx *app.AppContext) error {
	ctx.Logger.Info("repair dry-run started")
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)
	ctx.Logger.Info("invite provided: %t", hasValue(w.Invite))
	ctx.Logger.Info("dry-run only: no system changes will be made")
	addOperation(ctx, "report_directory", ctx.OutputDir, app.OperationStatusSuccess, "Report directory ready", "")

	plan, artifacts, err := w.Build(ctx)
	if err != nil {
		plan = failedPlan(ctx, err)
	}
	ctx.JSONValue = RepairPlanJSON{
		Command:        "repair",
		DryRun:         true,
		Status:         plan.Status,
		ExitCode:       exitCodeForPlanStatus(plan.Status),
		ReadyForRepair: plan.ReadyForRepair,
		ReportDir:      plan.ReportDir,
		RepairPlan:     plan,
	}
	ctx.ExitCode = exitCodeForPlanStatus(plan.Status)

	if artifacts.Detection != nil {
		if err := ctx.Reporter.WriteJSON("initial-detection", artifacts.Detection); err != nil {
			return err
		}
	}
	if artifacts.Classification != nil {
		if err := ctx.Reporter.WriteJSON("classification-result", artifacts.Classification); err != nil {
			return err
		}
	}
	if artifacts.Recommendation != nil {
		if err := ctx.Reporter.WriteJSON("recommendation-result", artifacts.Recommendation); err != nil {
			return err
		}
	}
	if artifacts.CleanupPlan != nil {
		if err := ctx.Reporter.WriteJSON("cleanup-plan", artifacts.CleanupPlan); err != nil {
			return err
		}
	}
	if plan.Preflight != nil {
		if err := ctx.Reporter.WriteJSON("preflight-result", plan.Preflight); err != nil {
			return err
		}
	}
	if plan.Installer.Validation != nil {
		if err := ctx.Reporter.WriteJSON("installer-validation", plan.Installer.Validation); err != nil {
			return err
		}
	}
	if err := ctx.Reporter.WriteJSON("repair-plan", plan); err != nil {
		return err
	}
	addOperation(ctx, "repair_plan_writing", "repair-plan.json", app.OperationStatusSuccess, "Repair plan written", "")

	summary := FormatDryRunSummary(plan, artifacts.Detection)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	addOperation(ctx, "summary_writing", "summary.txt", app.OperationStatusSuccess, "Dry-run summary written", "")
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatDryRunConsole(plan))
	}
	ctx.Logger.Info("repair dry-run completed with status: %s", plan.Status)
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)
	return nil
}

func (w DryRunWorkflow) Build(ctx *app.AppContext) (RepairPlan, repairPlanArtifacts, error) {
	started := time.Now()
	preflight := w.runPreflight(ctx)
	addOperation(ctx, "preflight", "repair", preflightOperationStatus(preflight), "Preflight checks completed", strings.Join(preflight.Errors, "; "))

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	initial := detect()
	addOperation(ctx, "initial_detection", "grabber", detectionStatus(initial), "Initial detection completed with health: "+string(initial.Health), "")

	classification := classifier.Classify(classifier.ClassificationInput{Detection: &initial})
	addOperation(ctx, "classification", "grabber", app.OperationStatusSuccess, "Issue classification completed", "")

	recommendationClassification := recommendations.FromClassifier(classification)
	recommendation := recommendations.Plan(recommendations.RecommendationInput{
		Detection:      &initial,
		Classification: &recommendationClassification,
		InstallerPath:  preflight.InstallerPath,
		HasInstaller:   preflight.InstallerAvailable,
		HasInvite:      hasValue(w.Invite),
		IsAdmin:        preflight.AdminRights,
		IsInteractive:  !ctx.NonInteractive && !ctx.Quiet,
		OutputDir:      ctx.OutputDir,
	})
	addOperation(ctx, "recommendation", "grabber", app.OperationStatusSuccess, "Recommendation plan generated", "")

	cleanupPlan := w.buildCleanupPlan(initial)
	cleanupPlan.Actions = cleaner.SortActionsForExecution(cleanupPlan.Actions)
	addPlanOperations(ctx, cleanupPlan)
	decision := Decide(initial, ctx.Force, len(cleanupPlan.Actions))

	defenderPlan := BuildDefenderPlan(initial)
	defenderPlan.Required = decision.DefenderNeeded
	defenderPlan.SystemModifying = decision.DefenderNeeded && defenderPlan.WouldAddExclusion
	addOperation(ctx, "defender_plan_generation", "defender", defenderPlanOperationStatus(defenderPlan), "Defender plan generated", strings.Join(defenderPlan.Warnings, "; "))

	installerPlan := BuildInstallerPlan(preflight, ctx.OutputDir, w.Invite)
	installerPlan.Required = decision.InstallNeeded
	installerPlan.WouldRunMsiexec = decision.InstallNeeded && installerPlan.InstallerFound
	addOperation(ctx, "installer_plan_generation", "installer", installerPlanOperationStatus(installerPlan), "Installer plan generated", strings.Join(installerPlan.Warnings, "; "))

	verificationPlan := BuildVerificationPlan()
	addOperation(ctx, "verification_plan_generation", "verification", app.OperationStatusSuccess, "Verification plan generated", "")

	plan := RepairPlan{
		Command:         "repair",
		DryRun:          true,
		Status:          RepairPlanStatusPlanned,
		ReadyForRepair:  true,
		ReportDir:       ctx.OutputDir,
		CreatedAt:       started.Format(time.RFC3339),
		Preflight:       &preflight,
		DetectionHealth: string(initial.Health),
		InstallMode:     string(initial.InstallMode),
		InstallRoot:     initial.InstallRoot,
		Cleanup:         buildCleanupPlanSummary(cleanupPlan, decision.CleanupNeeded),
		Defender:        defenderPlan,
		Installer:       installerPlan,
		Verification:    verificationPlan,
		Classification:  classification,
		Recommendation:  recommendation,
		ExpectedOutputs: expectedRepairOutputs(cleanupPlan),
		DryRunOutputs:   dryRunOutputs(),
		NextCommand:     NextRepairCommand(installerPlan.InstallerPath, !installerPlan.InstallerFound),
	}
	plan.RequiredInputs = requiredInputs(preflight)
	plan.Warnings = append(plan.Warnings, preflight.Warnings...)
	plan.Warnings = append(plan.Warnings, cleanupPlan.Warnings...)
	plan.Warnings = append(plan.Warnings, defenderPlan.Warnings...)
	plan.Warnings = append(plan.Warnings, installerPlan.Warnings...)
	plan.Errors = append(plan.Errors, preflight.Errors...)
	plan.Errors = append(plan.Errors, cleanupPlan.Blockers...)
	if len(plan.RequiredInputs) > 0 || preflight.HasFailedRequiredCheck() || len(cleanupPlan.Blockers) > 0 {
		plan.ReadyForRepair = false
		plan.Status = RepairPlanStatusNotReady
	} else if len(plan.Warnings) > 0 {
		plan.Status = RepairPlanStatusPlannedWithWarnings
	}

	return plan, repairPlanArtifacts{
		Detection:      &initial,
		Classification: &classification,
		Recommendation: &recommendation,
		CleanupPlan:    &cleanupPlan,
	}, nil
}

func (w DryRunWorkflow) runPreflight(ctx *app.AppContext) PreflightResult {
	isAdmin := checks.IsAdmin
	if w.IsAdmin != nil {
		isAdmin = w.IsAdmin
	}
	psCheck := checks.PowerShellAvailable
	if w.PowerShellCheck != nil {
		psCheck = w.PowerShellCheck
	}
	msiCheck := checks.MSIExecAvailable
	if w.MSIExecCheck != nil {
		msiCheck = w.MSIExecCheck
	}
	resolve := installer.ResolveInstaller
	if w.InstallerResolver != nil {
		resolve = w.InstallerResolver
	}
	return RunPreflight(PreflightOptions{
		Invite:            w.Invite,
		Installer:         w.Installer,
		ReportDir:         ctx.OutputDir,
		IsAdmin:           isAdmin,
		PowerShellCheck:   psCheck,
		MSIExecCheck:      msiCheck,
		InstallerResolver: resolve,
	})
}

func (w DryRunWorkflow) buildCleanupPlan(report detector.DetectionReport) cleaner.CleanupPlan {
	if w.BuildCleanupPlan != nil {
		return w.BuildCleanupPlan(report)
	}
	return cleaner.BuildPlan(report, cleaner.PlanOptions{DryRun: true})
}

type repairPlanArtifacts struct {
	Detection      *detector.DetectionReport
	Classification *classifier.ClassificationResult
	Recommendation *recommendations.RecommendationResult
	CleanupPlan    *cleaner.CleanupPlan
}

type RepairPlanJSON struct {
	Command        string     `json:"command"`
	DryRun         bool       `json:"dry_run"`
	Status         string     `json:"status"`
	ExitCode       int        `json:"exit_code"`
	ReadyForRepair bool       `json:"ready_for_repair"`
	ReportDir      string     `json:"report_dir"`
	RepairPlan     RepairPlan `json:"repair_plan"`
}

type RepairPlan struct {
	Command         string                               `json:"command"`
	DryRun          bool                                 `json:"dry_run"`
	Status          string                               `json:"status"`
	ReadyForRepair  bool                                 `json:"ready_for_repair"`
	ReportDir       string                               `json:"report_dir"`
	CreatedAt       string                               `json:"created_at"`
	Preflight       *PreflightResult                     `json:"preflight,omitempty"`
	DetectionHealth string                               `json:"detection_health,omitempty"`
	InstallMode     string                               `json:"install_mode,omitempty"`
	InstallRoot     string                               `json:"install_root,omitempty"`
	Cleanup         CleanupPlanSummary                   `json:"cleanup"`
	Defender        DefenderPlanSummary                  `json:"defender"`
	Installer       InstallerPlanSummary                 `json:"installer"`
	Verification    VerificationPlanSummary              `json:"verification"`
	Classification  classifier.ClassificationResult      `json:"classification,omitempty"`
	Recommendation  recommendations.RecommendationResult `json:"recommendation,omitempty"`
	ExpectedOutputs []string                             `json:"expected_outputs"`
	DryRunOutputs   []string                             `json:"dry_run_outputs"`
	RequiredInputs  []string                             `json:"required_inputs,omitempty"`
	Warnings        []string                             `json:"warnings,omitempty"`
	Errors          []string                             `json:"errors,omitempty"`
	NextCommand     string                               `json:"next_command,omitempty"`
}

type CleanupPlanSummary struct {
	Required     bool     `json:"required"`
	Destructive  bool     `json:"destructive"`
	PlanPath     string   `json:"plan_path,omitempty"`
	ActionsCount int      `json:"actions_count"`
	TargetsCount int      `json:"targets_count"`
	Actions      []string `json:"actions,omitempty"`
	WarningCount int      `json:"warning_count"`
}

type DefenderPlanSummary struct {
	Required           bool     `json:"required"`
	SystemModifying    bool     `json:"system_modifying"`
	RequiredPath       string   `json:"required_path,omitempty"`
	AlreadyCovered     bool     `json:"already_covered"`
	WouldAddExclusion  bool     `json:"would_add_exclusion"`
	ReadStatus         string   `json:"read_status,omitempty"`
	ExistingExclusions []string `json:"existing_exclusions,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

type InstallerPlanSummary struct {
	Required          bool                        `json:"required"`
	InstallerPath     string                      `json:"installer_path,omitempty"`
	InstallerFound    bool                        `json:"installer_found"`
	InstallerReadable bool                        `json:"installer_readable"`
	SHA256            string                      `json:"sha256,omitempty"`
	Validation        *installer.ValidationResult `json:"validation,omitempty"`
	Command           string                      `json:"command,omitempty"`
	WouldRunMsiexec   bool                        `json:"would_run_msiexec"`
	HasInvite         bool                        `json:"has_invite"`
	Warnings          []string                    `json:"warnings,omitempty"`
}

type VerificationPlanSummary struct {
	WouldRunAfterRepair bool     `json:"would_run_after_repair"`
	Checks              []string `json:"checks"`
}

type PreflightResult struct {
	Command             string                        `json:"command"`
	ReportDir           string                        `json:"report_dir"`
	AdminRights         bool                          `json:"admin_rights"`
	InstallerAvailable  bool                          `json:"installer_available"`
	InstallerPath       string                        `json:"installer_path,omitempty"`
	InstallerReadable   bool                          `json:"installer_readable"`
	InstallerResolution installer.InstallerResolution `json:"installer_resolution"`
	InstallerValidation *installer.ValidationResult   `json:"installer_validation,omitempty"`
	InvitePresent       bool                          `json:"invite_present"`
	MSIExecAvailable    bool                          `json:"msiexec_available"`
	PowerShellAvailable bool                          `json:"powershell_available"`
	Checks              []PreflightCheck              `json:"checks"`
	Warnings            []string                      `json:"warnings,omitempty"`
	Errors              []string                      `json:"errors,omitempty"`
	CreatedAt           time.Time                     `json:"created_at"`
}

type PreflightCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

type PreflightOptions struct {
	Invite            string
	Installer         string
	ReportDir         string
	IsAdmin           func() bool
	PowerShellCheck   func() (bool, error)
	MSIExecCheck      func() (bool, error)
	InstallerResolver func(string) (installer.InstallerResolution, error)
}

func RunPreflight(opts PreflightOptions) PreflightResult {
	result := PreflightResult{Command: "preflight", ReportDir: opts.ReportDir, CreatedAt: time.Now()}
	admin := false
	if opts.IsAdmin != nil {
		admin = opts.IsAdmin()
	}
	result.AdminRights = admin
	result.addCheck("admin_rights", admin, true, "Administrator rights are available", "Run PowerShell or CMD as Administrator.")

	result.InvitePresent = hasValue(opts.Invite)
	result.addCheck("invite_present", result.InvitePresent, true, "Invite is present", `Provide --invite "<INVITE>".`)

	resolve := opts.InstallerResolver
	if resolve == nil {
		resolve = installer.ResolveInstaller
	}
	resolution, err := resolve(opts.Installer)
	result.InstallerResolution = resolution
	result.InstallerValidation = installer.ValidationFromResolution(resolution)
	if err != nil {
		if result.InstallerValidation != nil {
			result.Warnings = append(result.Warnings, result.InstallerValidation.Warnings...)
			if !result.InstallerValidation.IsUsable() {
				result.Errors = append(result.Errors, result.InstallerValidation.ErrorSummary())
			}
		}
		result.Errors = append(result.Errors, err.Error())
		result.addCheck("installer_available", false, true, "", err.Error())
	} else {
		result.InstallerAvailable = true
		result.InstallerPath = resolution.SelectedPath
		validation := result.InstallerValidation
		result.InstallerValidation = validation
		if validation != nil {
			result.InstallerReadable = validation.Readable
			result.Warnings = append(result.Warnings, validation.Warnings...)
			if !validation.IsUsable() {
				result.InstallerAvailable = false
				result.Errors = append(result.Errors, validation.ErrorSummary())
				result.addCheck("installer_available", false, true, "", validation.ErrorSummary())
			} else {
				result.addCheck("installer_available", true, true, "Installer is available", "")
			}
		} else {
			result.InstallerReadable = false
			result.addCheck("installer_available", false, true, "", "installer validation did not run")
		}
	}

	msiAvailable, msiErr := runAvailabilityCheck(opts.MSIExecCheck)
	result.MSIExecAvailable = msiAvailable
	result.addCheck("msiexec_available", msiAvailable, true, "msiexec.exe is available", errorString(msiErr))

	psAvailable, psErr := runAvailabilityCheck(opts.PowerShellCheck)
	result.PowerShellAvailable = psAvailable
	result.addCheck("powershell_available", psAvailable, true, "powershell.exe is available", errorString(psErr))
	return result
}

func (r *PreflightResult) addCheck(name string, pass bool, required bool, message string, errText string) {
	status := "pass"
	if !pass {
		status = "fail"
		if errText == "" {
			errText = message
		}
	}
	check := PreflightCheck{Name: name, Status: status, Required: required, Message: message}
	if !pass {
		check.Error = errText
	}
	r.Checks = append(r.Checks, check)
	if !pass && required && errText != "" && name != "installer_available" {
		r.Errors = append(r.Errors, name+": "+errText)
	}
}

func (r PreflightResult) HasFailedRequiredCheck() bool {
	for _, check := range r.Checks {
		if check.Required && check.Status == "fail" {
			return true
		}
	}
	return false
}

func BuildCleanupPlanSummary(plan cleaner.CleanupPlan) CleanupPlanSummary {
	return buildCleanupPlanSummary(plan, len(plan.Actions) > 0)
}

func buildCleanupPlanSummary(plan cleaner.CleanupPlan, required bool) CleanupPlanSummary {
	actions := make([]string, 0, len(plan.Actions))
	targets := map[string]bool{}
	for _, action := range plan.Actions {
		actions = append(actions, string(action.Type)+": "+action.Target)
		targets[action.Target] = true
	}
	return CleanupPlanSummary{
		Required:     required && len(plan.Actions) > 0,
		Destructive:  required && len(plan.Actions) > 0,
		PlanPath:     "cleanup-plan.json",
		ActionsCount: len(plan.Actions),
		TargetsCount: len(targets),
		Actions:      actions,
		WarningCount: len(plan.Warnings),
	}
}

func BuildDefenderPlan(report detector.DetectionReport) DefenderPlanSummary {
	required, warnings := defender.RequiredPaths(report, false)
	plan := DefenderPlanSummary{
		Required:           len(required) > 0,
		SystemModifying:    len(required) > 0,
		ReadStatus:         "available",
		ExistingExclusions: append([]string{}, report.Defender.ExclusionPaths...),
		Warnings:           append([]string{}, warnings...),
	}
	if len(required) > 0 {
		plan.RequiredPath = required[0]
	}
	if !report.Defender.Available {
		plan.ReadStatus = "unavailable"
		if report.Defender.Error != "" {
			plan.Warnings = append(plan.Warnings, "Defender exclusions could not be verified: "+report.Defender.Error)
		} else {
			plan.Warnings = append(plan.Warnings, "Defender exclusions could not be verified")
		}
		return plan
	}
	plan.AlreadyCovered = len(required) > 0 && detector.IsPathCoveredByAnyExclusion(required[0], report.Defender.ExclusionPaths)
	plan.WouldAddExclusion = len(required) > 0 && !plan.AlreadyCovered
	return plan
}

func BuildInstallerPlan(preflight PreflightResult, reportDir string, invite string) InstallerPlanSummary {
	plan := InstallerPlanSummary{
		Required:          true,
		InstallerPath:     preflight.InstallerPath,
		InstallerFound:    preflight.InstallerAvailable,
		InstallerReadable: preflight.InstallerReadable,
		WouldRunMsiexec:   preflight.InstallerAvailable,
		HasInvite:         hasValue(invite),
		Validation:        preflight.InstallerValidation,
	}
	if preflight.InstallerValidation != nil {
		plan.SHA256 = preflight.InstallerValidation.SHA256
	}
	if plan.InstallerFound {
		plan.Command = fmt.Sprintf(`msiexec /i "%s" /qn /norestart invite=<REDACTED> /l*v "%s"`, plan.InstallerPath, filepath.Join(reportDir, "msi-install.log"))
	}
	return plan
}

func BuildVerificationPlan() VerificationPlanSummary {
	return VerificationPlanSummary{
		WouldRunAfterRepair: true,
		Checks: []string{
			"final health",
			"install root detected",
			"install mode detected",
			"primary service exists",
			"primary service running",
			"service executable exists",
			"expected Grabber process running if detectable",
			"Defender exclusion coverage",
			"MSI install log exists if install executed",
		},
	}
}

func NextRepairCommand(installerPath string, missingInstaller bool) string {
	if missingInstaller || strings.TrimSpace(installerPath) == "" {
		installerPath = `.\grabberEM.x64.msi`
	}
	return fmt.Sprintf(`.\kigrepair.exe repair --installer "%s" --invite "<INVITE>" --yes`, installerPath)
}

func failedPlan(ctx *app.AppContext, err error) RepairPlan {
	return RepairPlan{
		Command:         "repair",
		DryRun:          true,
		Status:          RepairPlanStatusFailed,
		ReadyForRepair:  false,
		ReportDir:       ctx.OutputDir,
		CreatedAt:       time.Now().Format(time.RFC3339),
		Errors:          []string{err.Error()},
		ExpectedOutputs: expectedRepairOutputs(cleaner.CleanupPlan{}),
		DryRunOutputs:   dryRunOutputs(),
		NextCommand:     NextRepairCommand("", true),
	}
}

func requiredInputs(preflight PreflightResult) []string {
	var inputs []string
	if !preflight.InstallerAvailable {
		inputs = append(inputs, "installer")
	}
	if !preflight.InvitePresent {
		inputs = append(inputs, "invite")
	}
	return inputs
}

func preflightOperationStatus(result PreflightResult) app.OperationStatus {
	if result.HasFailedRequiredCheck() {
		return app.OperationStatusFailed
	}
	if len(result.Warnings) > 0 {
		return app.OperationStatusWarning
	}
	return app.OperationStatusSuccess
}

func defenderPlanOperationStatus(plan DefenderPlanSummary) app.OperationStatus {
	if len(plan.Warnings) > 0 {
		return app.OperationStatusWarning
	}
	return app.OperationStatusSuccess
}

func installerPlanOperationStatus(plan InstallerPlanSummary) app.OperationStatus {
	if !plan.InstallerFound || !plan.HasInvite {
		return app.OperationStatusWarning
	}
	if len(plan.Warnings) > 0 {
		return app.OperationStatusWarning
	}
	return app.OperationStatusSuccess
}

func exitCodeForPlanStatus(status string) int {
	switch status {
	case RepairPlanStatusPlanned:
		return app.ExitSuccess
	case RepairPlanStatusPlannedWithWarnings:
		return app.ExitWarnings
	case RepairPlanStatusNotReady:
		return app.ExitVerificationFailed
	default:
		return app.ExitUnexpectedError
	}
}

func expectedRepairOutputs(plan cleaner.CleanupPlan) []string {
	outputs := []string{
		"initial-detection.json",
		"cleanup-plan.json",
		"defender-result.json",
		"install-result.json",
		"final-detection.json",
		"verification-result.json",
		"classification-result.json",
		"recommendation-result.json",
		"repair-result.json",
		"installer-validation.json",
		"operations.json",
		"summary.txt",
		"repair.log",
		"msi-install.log",
	}
	for _, action := range plan.Actions {
		if action.Type == cleaner.CleanupActionMSIUninstall {
			return append(outputs, "msi-uninstall.log")
		}
	}
	return outputs
}

func dryRunOutputs() []string {
	return []string{
		"repair-plan.json",
		"preflight-result.json",
		"initial-detection.json",
		"cleanup-plan.json",
		"classification-result.json",
		"recommendation-result.json",
		"installer-validation.json",
		"operations.json",
		"summary.txt",
		"repair.log",
	}
}

func runAvailabilityCheck(check func() (bool, error)) (bool, error) {
	if check == nil {
		return false, errors.New("availability check is not configured")
	}
	return check()
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func hasValue(value string) bool {
	return strings.TrimSpace(value) != ""
}
