package defender

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/config"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
)

const noInstallRootWarning = "No active installation root detected; Defender exclusions were not changed"
const noInstallRootRecommendation = "Run install or repair first, then run defender --ensure"

var (
	ErrDefenderConfirmationRequired = errors.New("Defender ensure in non-interactive mode requires --yes")
	ErrDefenderConfirmationDeclined = errors.New("defender ensure confirmation declined")
)

type DefenderWorkflow struct {
	Ensure        bool `json:"ensure"`
	Yes           bool `json:"yes,omitempty"`
	AllKnownPaths bool `json:"all_known_paths,omitempty"`

	Adder   ExclusionAdder
	Detect  func() detector.DetectionReport
	IsAdmin func() bool
	In      io.Reader
	Out     io.Writer
}

func (w DefenderWorkflow) Name() string {
	return "defender"
}

func (w DefenderWorkflow) Run(ctx *app.AppContext) error {
	startedAt := time.Now()
	result := DefenderEnsureResult{
		StartedAt:     startedAt,
		Ensure:        w.Ensure,
		AllKnownPaths: w.AllKnownPaths,
		ReportDir:     ctx.OutputDir,
	}
	ctx.Logger.Info("defender command started")
	ctx.Logger.Info("ensure mode: %t", w.Ensure)
	ctx.Logger.Info("all known paths mode: %t", w.AllKnownPaths)
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	initial := detect()
	result.InstallMode = string(initial.InstallMode)
	result.InstallRoot = initial.InstallRoot
	ctx.Logger.Info("admin status: %t", initial.IsAdmin)
	ctx.Logger.Info("detected install mode/root: %s / %s", initial.InstallMode, initial.InstallRoot)
	if err := ctx.Reporter.WriteJSON("initial-detection", initial); err != nil {
		return err
	}

	required, warnings := RequiredPaths(initial, w.AllKnownPaths && w.Ensure)
	result.RequiredPaths = required
	result.Warnings = append(result.Warnings, warnings...)
	result.ExistingExclusions = append([]string{}, initial.Defender.ExclusionPaths...)
	_, result.CoveredPaths, result.MissingBefore = detector.EvaluateDefenderCoverage(required, initial.Defender.ExclusionPaths)
	if !initial.Defender.Available {
		message := "Defender cmdlets unavailable"
		result.Warnings = append(result.Warnings, message)
		if initial.Defender.Error != "" {
			result.Errors = append(result.Errors, initial.Defender.Error)
		}
		status := app.OperationStatusWarning
		if w.Ensure {
			status = app.OperationStatusFailed
		}
		addOperation(ctx, "defender_status", "defender", status, message, initial.Defender.Error)
	}
	ctx.Logger.Info("required paths: %v", result.RequiredPaths)
	ctx.Logger.Info("existing exclusions count: %d", len(result.ExistingExclusions))
	ctx.Logger.Info("covered paths: %v", result.CoveredPaths)
	ctx.Logger.Info("missing paths before: %v", result.MissingBefore)

	addCoverageOperations(ctx, required, initial.Defender.ExclusionPaths, w.Ensure)

	if w.Ensure && (ctx.Quiet || ctx.NonInteractive) && !w.Yes {
		message := "Defender ensure in non-interactive mode requires --yes"
		ctx.ExitCode = ExitConfirmationRequired
		result.Errors = append(result.Errors, message)
		addOperation(ctx, "defender_confirmation", "user", app.OperationStatusFailed, message, message)
		ctx.Logger.Warn("confirmation status: missing")
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Fprintln(os.Stderr, message)
		}
		return w.finish(ctx, result, nil)
	}
	if w.Ensure {
		isAdmin := checks.IsAdmin
		if w.IsAdmin != nil {
			isAdmin = w.IsAdmin
		}
		if !isAdmin() {
			message := "Administrator rights are required to add Defender exclusions"
			ctx.ExitCode = ExitAdminRequired
			result.Errors = append(result.Errors, message)
			addOperation(ctx, "defender_admin", "administrator", app.OperationStatusFailed, message, message)
			ctx.Logger.Error(message)
			if !ctx.Quiet && !ctx.JSONOutput {
				fmt.Fprintln(os.Stderr, message)
			}
			return w.finish(ctx, result, nil)
		}
	}

	if len(required) == 0 && !w.AllKnownPaths {
		addOperation(ctx, "defender_required_paths", "grabber", app.OperationStatusWarning, noInstallRootWarning, "")
		ctx.ExitCode = ExitNoInstallationRoot
		return w.finish(ctx, result, nil)
	}
	if !initial.Defender.Available {
		if w.Ensure {
			ctx.ExitCode = ExitAddFailed
		} else {
			ctx.ExitCode = ExitWarnings
		}
		return w.finish(ctx, result, nil)
	}
	if !w.Ensure {
		if len(result.MissingBefore) > 0 {
			ctx.ExitCode = ExitWarnings
		}
		return w.finish(ctx, result, nil)
	}
	if len(result.MissingBefore) == 0 {
		ctx.ExitCode = ExitOK
		final := detect()
		_ = ctx.Reporter.WriteJSON("final-detection", final)
		return w.finish(ctx, result, &final)
	}

	if !w.Yes {
		if !ctx.Quiet && !ctx.JSONOutput {
			fmt.Print(FormatPreflight(initial, result, ctx.OutputDir))
		}
		if err := ConfirmEnsure(ctx, w.Yes, inputOrStdin(w.In), outputOrStdout(w.Out)); err != nil {
			ctx.ExitCode = ExitConfirmationRequired
			message := err.Error()
			if errors.Is(err, ErrDefenderConfirmationRequired) {
				message = "Defender ensure in non-interactive mode requires --yes"
			}
			result.Errors = append(result.Errors, message)
			addOperation(ctx, "defender_confirmation", "user", app.OperationStatusFailed, message, message)
			ctx.Logger.Warn("confirmation status: declined or missing")
			if !ctx.Quiet && !ctx.JSONOutput {
				fmt.Fprintln(os.Stderr, message)
			}
			return w.finish(ctx, result, nil)
		}
	}
	ctx.Logger.Info("confirmation status: accepted")

	adder := w.Adder
	if adder == nil {
		adder = PowerShellExclusionAdder{}
	}
	for _, path := range result.MissingBefore {
		ctx.Logger.Info("attempting Defender exclusion add: %s", path)
		ctx.Logger.Info("executing command: powershell.exe -NoProfile -ExecutionPolicy Bypass -Command Add-MpPreference -ExclusionPath <path>")
		commandResult := adder.AddExclusion(path)
		ctx.Logger.Info("Add-MpPreference exit code for %s: %d", path, commandResult.ExitCode)
		if commandResult.ExitCode == 0 && commandResult.Err == nil {
			result.AddedPaths = append(result.AddedPaths, path)
			addOperation(ctx, "defender_exclusion_add", path, app.OperationStatusSuccess, "Defender exclusion added", "")
			continue
		}
		errText := commandErrorSummary(commandResult)
		result.FailedPaths = append(result.FailedPaths, path)
		result.Errors = append(result.Errors, errText)
		addOperation(ctx, "defender_exclusion_add", path, app.OperationStatusFailed, "Defender exclusion add failed", errText)
	}

	final := detect()
	if err := ctx.Reporter.WriteJSON("final-detection", final); err != nil {
		return err
	}
	result.MissingAfter = missingPaths(required, final.Defender.ExclusionPaths)
	ctx.Logger.Info("missing paths after: %v", result.MissingAfter)
	if len(result.FailedPaths) > 0 {
		ctx.ExitCode = ExitAddFailed
	} else if len(result.MissingAfter) > 0 {
		ctx.ExitCode = ExitVerificationFailed
		result.Errors = append(result.Errors, "Defender exclusion verification failed")
		addOperation(ctx, "defender_verify", "defender-exclusions", app.OperationStatusFailed, "Verification failed after adding Defender exclusions", strings.Join(result.MissingAfter, "; "))
	} else {
		ctx.ExitCode = ExitOK
		addOperation(ctx, "defender_verify", "defender-exclusions", app.OperationStatusSuccess, "All required Defender exclusions are covered", "")
	}
	return w.finish(ctx, result, &final)
}

func RequiredPaths(report detector.DetectionReport, allKnown bool) ([]string, []string) {
	if allKnown {
		return ExpandedDefenderPaths(), nil
	}
	switch report.InstallMode {
	case detector.InstallModeStandard, detector.InstallModeHelper:
		if strings.TrimSpace(report.InstallRoot) != "" {
			return []string{detector.NormalizeWindowsPath(report.InstallRoot)}, nil
		}
	case detector.InstallModeHiddenWMI:
		if strings.TrimSpace(report.System.SystemRoot) != "" {
			return []string{detector.NormalizeWindowsPath(filepath.Join(report.System.SystemRoot, "System32", "wmi"))}, nil
		}
		if strings.TrimSpace(report.InstallRoot) != "" {
			return []string{detector.NormalizeWindowsPath(report.InstallRoot)}, nil
		}
	}
	return nil, []string{noInstallRootWarning, noInstallRootRecommendation}
}

func ExpandedDefenderPaths() []string {
	paths := make([]string, 0, len(config.DefenderExclusionPaths))
	for _, path := range config.DefenderExclusionPaths {
		paths = append(paths, detector.NormalizeWindowsPath(config.ExpandPath(path)))
	}
	return paths
}

func ConfirmEnsure(ctx *app.AppContext, yes bool, in io.Reader, out io.Writer) error {
	if yes {
		return nil
	}
	if ctx.Quiet || ctx.NonInteractive {
		return ErrDefenderConfirmationRequired
	}
	if out != nil {
		fmt.Fprint(out, "Add missing Defender exclusions? Type YES to continue: ")
	}
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return ErrDefenderConfirmationDeclined
	}
	if scanner.Text() != "YES" {
		return ErrDefenderConfirmationDeclined
	}
	return nil
}

func (w DefenderWorkflow) finish(ctx *app.AppContext, result DefenderEnsureResult, final *detector.DetectionReport) error {
	result.FinishedAt = time.Now()
	result.ExitCode = ctx.ExitCode
	ctx.JSONValue = result
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)
	if err := ctx.Reporter.WriteJSON("defender-result", result); err != nil {
		return err
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

func addCoverageOperations(ctx *app.AppContext, required []string, exclusions []string, ensure bool) {
	for _, path := range required {
		if detector.IsPathCoveredByAnyExclusion(path, exclusions) {
			addOperation(ctx, "defender_exclusion_check", path, app.OperationStatusSuccess, "Path is already covered by Defender exclusion", "")
			continue
		}
		if !ensure {
			addOperation(ctx, "defender_exclusion_check", path, app.OperationStatusWarning, "Path is missing Defender exclusion", "")
		}
	}
}

func addOperation(ctx *app.AppContext, step string, target string, status app.OperationStatus, message string, errText string) {
	result := app.OperationResult{
		Step:      step,
		Target:    target,
		Status:    status,
		Message:   message,
		Error:     errText,
		Timestamp: time.Now(),
	}
	ctx.AddResult(result)
	logging.LogOperation(ctx.Logger, result)
}

func missingPaths(required, exclusions []string) []string {
	_, _, missing := detector.EvaluateDefenderCoverage(required, exclusions)
	return missing
}

func inputOrStdin(in io.Reader) io.Reader {
	if in != nil {
		return in
	}
	return os.Stdin
}

func outputOrStdout(out io.Writer) io.Writer {
	if out != nil {
		return out
	}
	return os.Stdout
}
