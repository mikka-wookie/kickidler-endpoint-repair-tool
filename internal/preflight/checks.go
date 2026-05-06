package preflight

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/config"
	"kigrepair/internal/detector"
	"kigrepair/internal/installer"
	"kigrepair/internal/logging"
	"kigrepair/internal/version"
	"kigrepair/internal/winapi"
)

func (w Workflow) Name() string {
	return "preflight"
}

func (w Workflow) Run(ctx *app.AppContext) error {
	result, err := Run(ctx, Options{
		InstallerPath: w.InstallerPath,
		HasInvite:     w.HasInvite,
		OutputDir:     ctx.OutputDir,
		JSON:          ctx.JSONOutput,
		Quiet:         ctx.Quiet,
	}, w.Deps)
	if err != nil {
		ctx.ExitCode = app.ExitUnexpectedError
		return err
	}
	ctx.ExitCode = result.ExitCode
	ctx.JSONValue = result
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(FormatConsoleSummary(result))
	}
	return nil
}

func Run(ctx *app.AppContext, opts Options, deps Dependencies) (Result, error) {
	deps = fillDependencies(deps)
	now := deps.Now()
	result := Result{
		Build:     version.Get(),
		Command:   "preflight",
		ReportDir: opts.OutputDir,
		CreatedAt: now.Format(time.RFC3339),
		Checks:    []CheckResult{},
		Warnings:  []string{},
		Errors:    []string{},
	}
	addOperation(ctx, "preflight.report_directory", opts.OutputDir, app.OperationStatusSuccess, "Report directory created", "")

	env := EnvironmentCheck{}
	env.IsAdmin = deps.IsAdmin()
	env.OS = deps.OS()
	env.Architecture = deps.Architecture()
	if exePath, err := deps.ExePath(); err == nil {
		env.ExePath = exePath
	}
	if workDir, err := deps.WorkingDir(); err == nil {
		env.WorkingDir = workDir
	}
	if _, err := deps.LookPath("powershell.exe"); err == nil {
		env.PowerShellFound = true
	} else if _, err := deps.LookPath("powershell"); err == nil {
		env.PowerShellFound = true
	}
	if _, err := deps.LookPath("msiexec.exe"); err == nil {
		env.MsiexecFound = true
	} else if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		if _, err := os.Stat(filepath.Join(systemRoot, "System32", "msiexec.exe")); err == nil {
			env.MsiexecFound = true
		}
	}
	result.Environment = env

	addCheck(&result, CheckResult{Code: "admin_rights", Required: true, Title: "Administrator rights", Status: passFail(env.IsAdmin), Evidence: boolEvidence(env.IsAdmin), Action: failAction(env.IsAdmin, "Run PowerShell or CMD as Administrator.")})
	reportWriteErr := deps.WriteProbe(opts.OutputDir)
	addCheck(&result, CheckResult{Code: "report_directory_writable", Required: true, Title: "Report directory writable", Status: statusFromErr(reportWriteErr), Description: errorString(reportWriteErr), Evidence: opts.OutputDir, Action: "Check permissions for the report output path."})
	addCheck(&result, CheckResult{Code: "windows_os", Required: true, Title: "Windows OS", Status: passFail(strings.EqualFold(env.OS, "windows")), Evidence: env.OS, Action: "Run kigrepair on a supported Windows endpoint."})
	addCheck(&result, CheckResult{Code: "architecture_detected", Required: true, Title: "Architecture detected", Status: passFail(strings.TrimSpace(env.Architecture) != ""), Evidence: env.Architecture, Action: "Run from a supported Go/Windows runtime."})
	addCheck(&result, CheckResult{Code: "msiexec_available", Required: true, Title: "msiexec available", Status: passFail(env.MsiexecFound), Action: "Restore Windows Installer or verify msiexec.exe is available."})
	addCheck(&result, CheckResult{Code: "powershell_available", Required: true, Title: "PowerShell available", Status: passFail(env.PowerShellFound), Action: "Install or repair Windows PowerShell."})

	resolution := deps.ResolveInstaller(opts.InstallerPath)
	result.Installer = checkInstaller(resolution, deps)
	addOperation(ctx, "preflight.installer_discovery", "installer", installerOperationStatus(result.Installer), installerOperationMessage(result.Installer), result.Installer.Error)
	addCheck(&result, CheckResult{Code: "installer_available", Required: true, Title: "Installer available", Status: installerAvailableStatus(result.Installer), Evidence: result.Installer.Path, Action: "Provide --installer \"C:\\Path\\grabberEM.x64.msi\" or place a supported installer next to kigrepair.exe."})
	addCheck(&result, CheckResult{Code: "installer_supported_name", Required: false, Title: "Installer filename is recognized", Status: supportedNameStatus(result.Installer), Evidence: filepath.Base(result.Installer.Path), Action: "Use grabberEM.x64.msi, grabberEM.x32.msi, grabberTT.x64.msi, grabberTT.x32.msi, or grabber.msi when possible."})
	addCheck(&result, CheckResult{Code: "invite_present", Required: true, Title: "Invite present", Status: passFail(opts.HasInvite), Evidence: boolEvidence(opts.HasInvite), Action: failAction(opts.HasInvite, "Provide --invite \"<INVITE>\".")})

	var detection detector.DetectionReport
	detection, detectErr := deps.Detect()
	if detectErr != nil {
		addCheck(&result, CheckResult{Code: "detection_completed", Required: false, Title: "Detection completed", Status: CheckWarning, Description: detectErr.Error(), Action: "Review repair.log and run check for more details."})
		addOperation(ctx, "preflight.detection", "grabber", app.OperationStatusWarning, "Detection failed or was incomplete", detectErr.Error())
	} else {
		result.DetectionHealth = string(detection.Health)
		result.InstallMode = string(detection.InstallMode)
		result.InstallRoot = detection.InstallRoot
		env.DefenderReadable = detection.Defender.Available
		result.Environment = env
		addCheck(&result, serviceQueryCheck(detection))
		addCheck(&result, registryQueryCheck(detection))
		addCheck(&result, defenderReadCheck(detection))
		addCheck(&result, CheckResult{Code: "detection_completed", Required: false, Title: "Detection completed", Status: CheckPass, Evidence: string(detection.Health)})
		addOperation(ctx, "preflight.detection", "grabber", app.OperationStatusSuccess, "Detection completed with health: "+string(detection.Health), "")
		if err := ctx.Reporter.WriteJSON("initial-detection", detection); err != nil {
			return result, err
		}
	}

	result.Classification = Classify(detection, detectErr)
	result.Recommendation = Recommend(result, detection)
	result.Policy = ctx.ConfigPolicy()
	result.PolicyWarnings, result.BlockingPolicyChecks = policyReadiness(result.Policy, result, detection)
	for _, blocker := range result.BlockingPolicyChecks {
		addCheck(&result, CheckResult{Code: blocker, Required: true, Title: "Policy gate", Status: CheckFail, Description: blocker})
	}
	result.Status, result.ReadyForRepair = calculateStatus(result.Checks)
	result.ExitCode = ExitCode(result.Status)
	result.FailedRequiredChecks = failedRequiredChecks(result.Checks)
	result.Warnings = warningCodes(result.Checks)
	result.Errors = failedRequiredChecks(result.Checks)

	addOperation(ctx, "preflight.classification", "grabber", app.OperationStatusSuccess, "Classification completed", "")
	addOperation(ctx, "preflight.recommendation", "next-action", app.OperationStatusSuccess, "Recommendation completed", "")
	if err := ctx.Reporter.WriteJSON("classification-result", result.Classification); err != nil {
		return result, err
	}
	if err := ctx.Reporter.WriteJSON("recommendation-result", result.Recommendation); err != nil {
		return result, err
	}
	if err := ctx.Reporter.WriteJSON("preflight-result", result); err != nil {
		return result, err
	}
	addOperation(ctx, "preflight.report_writing", "preflight-result.json", app.OperationStatusSuccess, "Preflight reports written", "")
	if err := ctx.Reporter.WriteText("summary", FormatSummary(result)); err != nil {
		return result, err
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return result, err
	}
	ctx.Logger.Info("preflight status: %s", result.Status)
	ctx.Logger.Info("preflight exit code: %d", result.ExitCode)
	return result, nil
}

func policyReadiness(policy config.PolicySummary, result Result, detection detector.DetectionReport) ([]string, []string) {
	warnings := []string{"Active policy profile: " + policy.Profile}
	blockers := []string{}
	if policy.RequireRollbackSnapshot {
		warnings = append(warnings, "rollback_required")
	}
	if policy.RequireInstallerValidation {
		warnings = append(warnings, "installer_validation_required")
	}
	if !policy.AllowRealRepair {
		blockers = append(blockers, "repair_disabled_by_policy")
	}
	if policy.StopOnDetectionUnknown && strings.EqualFold(string(detection.Health), string(detector.GrabberHealthUnknown)) {
		blockers = append(blockers, "unknown_detection_blocks_repair")
	}
	if policy.Profile == "conservative" && policy.RequireDefenderBeforeInstall && policy.DefenderRequireCoverageForInstall && !result.Environment.DefenderReadable && len(detection.MissingDefenderPaths) > 0 && !policy.DefenderTreatUnavailableAsWarning {
		blockers = append(blockers, "defender_required_but_unavailable")
	}
	if policy.Profile == "diagnostic" {
		warnings = append(warnings, "diagnostic_profile_recommends_collect_report_before_repair")
	}
	return warnings, blockers
}

func fillDependencies(deps Dependencies) Dependencies {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.IsAdmin == nil {
		deps.IsAdmin = winapi.IsAdmin
	}
	if deps.OS == nil {
		deps.OS = func() string { return runtime.GOOS }
	}
	if deps.Architecture == nil {
		deps.Architecture = func() string { return runtime.GOARCH }
	}
	if deps.ExePath == nil {
		deps.ExePath = os.Executable
	}
	if deps.WorkingDir == nil {
		deps.WorkingDir = os.Getwd
	}
	if deps.LookPath == nil {
		deps.LookPath = exec.LookPath
	}
	if deps.ResolveInstaller == nil {
		deps.ResolveInstaller = defaultResolveInstaller
	}
	if deps.Detect == nil {
		deps.Detect = func() (detector.DetectionReport, error) { return detector.Detect(), nil }
	}
	if deps.ReadFile == nil {
		deps.ReadFile = readableFile
	}
	if deps.HashFile == nil {
		deps.HashFile = hashFile
	}
	if deps.WriteProbe == nil {
		deps.WriteProbe = writeProbe
	}
	return deps
}

func defaultResolveInstaller(path string) InstallerResolution {
	resolution, err := installer.ResolveInstaller(path)
	result := InstallerResolution{
		Path:       resolution.SelectedPath,
		Provided:   strings.TrimSpace(path) != "",
		Discovered: strings.TrimSpace(path) == "" && resolution.SelectedPath != "",
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func checkInstaller(resolution InstallerResolution, deps Dependencies) InstallerCheck {
	check := InstallerCheck{
		Required:   true,
		Provided:   resolution.Provided,
		Discovered: resolution.Discovered,
		Path:       resolution.Path,
		Error:      resolution.Error,
	}
	if strings.TrimSpace(check.Path) == "" {
		return check
	}
	if _, _, ok := installer.ParseInstallerName(filepath.Base(check.Path)); ok {
		check.SupportedName = true
	}
	if err := deps.ReadFile(check.Path); err != nil {
		check.Exists = !errors.Is(err, os.ErrNotExist)
		check.Readable = false
		if check.Error == "" {
			check.Error = err.Error()
		}
		return check
	}
	check.Exists = true
	check.Readable = true
	if sum, err := deps.HashFile(check.Path); err == nil {
		check.SHA256 = sum
	} else if check.Error == "" {
		check.Error = "SHA-256 calculation failed: " + err.Error()
	}
	return check
}

func readableFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeProbe(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, ".preflight-write-test")
	if err := os.WriteFile(path, []byte("ok"), 0600); err != nil {
		return err
	}
	_ = os.Remove(path)
	return nil
}

func serviceQueryCheck(report detector.DetectionReport) CheckResult {
	errorsFound := []string{}
	for _, service := range report.Services {
		if strings.TrimSpace(service.Error) != "" {
			errorsFound = append(errorsFound, service.Name+": "+service.Error)
		}
	}
	if len(errorsFound) > 0 {
		return CheckResult{Code: "service_query", Status: CheckWarning, Required: false, Title: "Service query", Description: strings.Join(errorsFound, "; "), Action: "Run from an account allowed to query Windows services."}
	}
	return CheckResult{Code: "service_query", Status: CheckPass, Required: false, Title: "Service query", Evidence: fmt.Sprintf("%d supported service(s) checked", len(report.Services))}
}

func registryQueryCheck(report detector.DetectionReport) CheckResult {
	errorsFound := []string{}
	for _, key := range report.Registry {
		if strings.TrimSpace(key.Error) != "" {
			errorsFound = append(errorsFound, key.Root+`\`+key.Path+": "+key.Error)
		}
	}
	if len(errorsFound) > 0 {
		return CheckResult{Code: "registry_query", Status: CheckWarning, Required: false, Title: "Registry query", Description: strings.Join(errorsFound, "; "), Action: "Run from an account allowed to read the known registry hives."}
	}
	return CheckResult{Code: "registry_query", Status: CheckPass, Required: false, Title: "Registry query", Evidence: fmt.Sprintf("%d key(s) checked", len(report.Registry))}
}

func defenderReadCheck(report detector.DetectionReport) CheckResult {
	if report.Defender.Available {
		return CheckResult{Code: "defender_read", Status: CheckPass, Required: false, Title: "Defender status readable", Evidence: fmt.Sprintf("%d exclusion path(s)", len(report.Defender.ExclusionPaths))}
	}
	return CheckResult{Code: "defender_read", Status: CheckWarning, Required: false, Title: "Defender status readable", Description: report.Defender.Error, Action: "Repair can continue, but verify Defender status manually if exclusions cannot be read."}
}

func Classify(report detector.DetectionReport, err error) ClassificationResult {
	if err != nil {
		issue := ClassificationIssue{Code: "detection_failed", Message: err.Error()}
		return ClassificationResult{Status: "unknown", PrimaryIssue: &issue, Issues: []ClassificationIssue{issue}}
	}
	result := ClassificationResult{Status: "no_issue_detected", Health: string(report.Health)}
	if report.Health == detector.GrabberHealthHealthy {
		return result
	}
	result.Status = "issue_detected"
	issue := issueFromDetection(report)
	result.PrimaryIssue = &issue
	result.Issues = append(result.Issues, issue)
	return result
}

func issueFromDetection(report detector.DetectionReport) ClassificationIssue {
	if report.PrimaryService != "" && report.ServiceExecutablePath != "" && !report.ServiceExecutableExists {
		return ClassificationIssue{Code: "service_binary_missing", Message: "Known service exists but the service executable is missing."}
	}
	if report.Health == detector.GrabberHealthPartiallyRemoved {
		return ClassificationIssue{Code: "partial_removal_detected", Message: "Grabber leftovers are present without a healthy service."}
	}
	if report.Health == detector.GrabberHealthNotInstalled {
		return ClassificationIssue{Code: "not_installed", Message: "No known Grabber installation was detected."}
	}
	if len(report.MissingDefenderPaths) > 0 {
		return ClassificationIssue{Code: "defender_exclusion_missing", Message: "Required Defender exclusion is missing."}
	}
	if len(report.Issues) > 0 {
		return ClassificationIssue{Code: "detection_issue", Message: report.Issues[0]}
	}
	return ClassificationIssue{Code: "unknown_health", Message: "Detection did not produce a specific issue code."}
}

func Recommend(result Result, report detector.DetectionReport) RecommendationResult {
	command := RepairCommand(result.Installer.Path)
	action := RecommendationAction{Code: "run_full_repair", Message: "Run the full repair workflow after failed required preflight checks are fixed.", Command: command}
	rec := RecommendationResult{PrimaryAction: &action, Actions: []RecommendationAction{action}}
	if result.ReadyForRepair || len(failedRequiredChecks(result.Checks)) == 0 {
		rec.Status = "ready"
		rec.PrimaryAction.Message = "Run full repair."
		return rec
	}
	rec.Status = "missing_required_input"
	rec.Actions = append([]RecommendationAction{{
		Code:    "fix_preflight_checks",
		Message: "Fix failed required preflight checks before running repair.",
	}}, rec.Actions...)
	return rec
}

func RepairCommand(installerPath string) string {
	if strings.TrimSpace(installerPath) == "" {
		return `.\kigrepair.exe repair --installer "C:\Path\grabberEM.x64.msi" --invite "<INVITE>" --yes`
	}
	return `.\kigrepair.exe repair --installer "` + strings.ReplaceAll(installerPath, `"`, `\"`) + `" --invite "<INVITE>" --yes`
}

func calculateStatus(checks []CheckResult) (string, bool) {
	failedRequired := false
	warnings := false
	for _, check := range checks {
		if check.Required && check.Status == CheckFail {
			failedRequired = true
		}
		if check.Status == CheckWarning {
			warnings = true
		}
	}
	if failedRequired {
		return StatusNotReady, false
	}
	if warnings {
		return StatusReadyWithWarnings, true
	}
	return StatusReady, true
}

func ExitCode(status string) int {
	switch status {
	case StatusReady:
		return app.ExitSuccess
	case StatusReadyWithWarnings:
		return app.ExitWarnings
	case StatusNotReady:
		return app.ExitVerificationFailed
	default:
		return app.ExitUnexpectedError
	}
}

func addCheck(result *Result, check CheckResult) {
	if check.Status == CheckPass {
		check.Action = ""
	}
	if check.Status == CheckFail && check.Description == "" {
		check.Description = check.Action
	}
	result.Checks = append(result.Checks, check)
}

func passFail(ok bool) string {
	if ok {
		return CheckPass
	}
	return CheckFail
}

func statusFromErr(err error) string {
	if err == nil {
		return CheckPass
	}
	return CheckFail
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func failAction(ok bool, action string) string {
	if ok {
		return ""
	}
	return action
}

func boolEvidence(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func installerAvailableStatus(check InstallerCheck) string {
	if check.Exists && check.Readable {
		return CheckPass
	}
	return CheckFail
}

func supportedNameStatus(check InstallerCheck) string {
	if strings.TrimSpace(check.Path) == "" || !check.Exists {
		return CheckSkipped
	}
	if check.SupportedName {
		return CheckPass
	}
	return CheckWarning
}

func installerOperationStatus(check InstallerCheck) app.OperationStatus {
	if check.Exists && check.Readable {
		return app.OperationStatusSuccess
	}
	return app.OperationStatusFailed
}

func installerOperationMessage(check InstallerCheck) string {
	if check.Exists && check.Readable {
		if check.Discovered {
			return "Installer discovered: " + check.Path
		}
		return "Installer validated: " + check.Path
	}
	return "Installer missing or unreadable"
}

func failedRequiredChecks(checks []CheckResult) []string {
	result := []string{}
	for _, check := range checks {
		if check.Required && check.Status == CheckFail {
			result = append(result, check.Code)
		}
	}
	return result
}

func warningCodes(checks []CheckResult) []string {
	result := []string{}
	for _, check := range checks {
		if check.Status == CheckWarning {
			result = append(result, check.Code)
		}
	}
	return result
}

func addOperation(ctx *app.AppContext, step string, target string, status app.OperationStatus, message string, errorText string) {
	if ctx == nil {
		return
	}
	result := app.OperationResult{Step: step, Target: target, Status: status, Message: message, Error: errorText, Timestamp: time.Now()}
	ctx.AddResult(result)
	if ctx.Logger != nil {
		logging.LogOperation(ctx.Logger, result)
	}
}
