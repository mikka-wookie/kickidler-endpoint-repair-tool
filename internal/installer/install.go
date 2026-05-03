package installer

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
)

type InstallWorkflow struct {
	Invite    string `json:"-"`
	Installer string `json:"installer,omitempty"`
	Yes       bool   `json:"yes,omitempty"`

	Executor MSIExecutor
	Detect   func() detector.DetectionReport
	IsAdmin  func() bool
}

func (w InstallWorkflow) Name() string {
	return "install"
}

func (w InstallWorkflow) Run(ctx *app.AppContext) error {
	startedAt := time.Now()
	result := InstallResult{
		StartedAt:      startedAt,
		InviteProvided: strings.TrimSpace(w.Invite) != "",
		MSI:            MSIResult{ExitCode: -1, Status: "not_run", Message: "MSI install was not run"},
		ReportDir:      ctx.OutputDir,
	}
	ctx.Logger.Info("install started")
	ctx.Logger.Info("report directory: %s", ctx.OutputDir)
	ctx.Logger.Info("invite provided: %t", result.InviteProvided)

	invite, err := ValidateInvite(w.Invite)
	if err != nil {
		return w.finishEarly(ctx, result, ExitInstallInvalidInvite, "install.validate_invite", "invite", "Invalid invite", err)
	}

	installerPath, err := ValidateInstallerPath(w.Installer)
	if err != nil {
		return w.finishEarly(ctx, result, ExitInstallInvalidInstaller, "install.validate_installer", "installer", "Invalid installer", err)
	}
	result.InstallerPath = installerPath
	ctx.Logger.Info("installer path: %s", installerPath)

	isAdmin := checks.IsAdmin
	if w.IsAdmin != nil {
		isAdmin = w.IsAdmin
	}
	admin := isAdmin()
	ctx.Logger.Info("admin status: %t", admin)
	if !admin {
		message := "Administrator rights are required for install"
		return w.finishEarly(ctx, result, ExitInstallAdminRequired, "install.admin", "administrator", message, errors.New(message))
	}

	detect := detector.Detect
	if w.Detect != nil {
		detect = w.Detect
	}
	initial := detect()
	result.InitialHealth = string(initial.Health)
	ctx.Logger.Info("initial health: %s", initial.Health)
	if err := ctx.Reporter.WriteJSON("initial-detection", initial); err != nil {
		return err
	}
	initialResult := app.OperationResult{
		Step:      "install.initial_detection",
		Target:    "grabber",
		Status:    detectionOperationStatus(initial),
		Message:   "Initial detection completed with health: " + string(initial.Health),
		Timestamp: time.Now(),
	}
	ctx.AddResult(initialResult)
	logging.LogOperation(ctx.Logger, initialResult)

	msiLogPath := filepath.Join(ctx.OutputDir, "msi-install.log")
	executor := w.Executor
	if executor == nil {
		executor = ExecMSIExecutor{}
	}
	ctx.Logger.Info("msiexec started")
	ctx.Logger.Info("executing command: msiexec.exe %s", strings.Join(MaskedMSIInstallArgs(installerPath, msiLogPath), " "))
	commandResult := executor.Install(installerPath, invite, msiLogPath)
	msi := ClassifyMSIInstallExitCode(commandResult.ExitCode)
	result.MSI = msi
	ctx.Logger.Info("msiexec exit code: %d", commandResult.ExitCode)
	ctx.Logger.Info("MSI classified status: %s", msi.Status)

	msiOperation := app.OperationResult{
		Step:      "msi_install",
		Target:    installerPath,
		Status:    msiOperationStatus(msi),
		Message:   fmt.Sprintf("%s; exit code %d", msi.Status, msi.ExitCode),
		Timestamp: time.Now(),
	}
	if !msi.Success {
		msiOperation.Error = msi.Message
	}
	ctx.AddResult(msiOperation)
	logging.LogOperation(ctx.Logger, msiOperation)

	final := detect()
	result.FinalHealth = string(final.Health)
	result.FinalInstallMode = string(final.InstallMode)
	result.FinalInstallRoot = final.InstallRoot
	ctx.Logger.Info("final health: %s", final.Health)
	if err := ctx.Reporter.WriteJSON("final-detection", final); err != nil {
		return err
	}

	verification := VerifyInstall(msi, final)
	result.Warnings = append(result.Warnings, verification.Warnings...)
	result.Errors = append(result.Errors, verification.Errors...)
	ctx.Logger.Info("final verification result: %s", verification.Status)
	verifyOperation := app.OperationResult{
		Step:      "install.verify",
		Target:    "grabber",
		Status:    verificationOperationStatus(verification),
		Message:   verification.Message,
		Timestamp: time.Now(),
	}
	if len(verification.Errors) > 0 {
		verifyOperation.Error = strings.Join(verification.Errors, "; ")
	}
	ctx.AddResult(verifyOperation)
	logging.LogOperation(ctx.Logger, verifyOperation)

	ctx.ExitCode = installExitCode(msi, verification)
	result.ExitCode = ctx.ExitCode
	result.FinishedAt = time.Now()
	ctx.JSONValue = result
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)

	if err := ctx.Reporter.WriteJSON("install-result", result); err != nil {
		return err
	}
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	summary := FormatInstallSummary(result, &final)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func (w InstallWorkflow) finishEarly(ctx *app.AppContext, result InstallResult, code int, step string, target string, message string, err error) error {
	result.Errors = append(result.Errors, err.Error())
	result.ExitCode = code
	result.FinishedAt = time.Now()
	ctx.ExitCode = code
	ctx.JSONValue = result
	status := app.OperationStatusFailed
	operation := app.OperationResult{
		Step:      step,
		Target:    target,
		Status:    status,
		Message:   message,
		Error:     err.Error(),
		Timestamp: time.Now(),
	}
	ctx.AddResult(operation)
	logging.LogOperation(ctx.Logger, operation)
	if code == ExitInstallAdminRequired && !ctx.Quiet && !ctx.JSONOutput {
		fmt.Println("Administrator rights are required for install")
	}
	ctx.Logger.Info("final exit code: %d", ctx.ExitCode)
	if writeErr := ctx.Reporter.WriteJSON("install-result", result); writeErr != nil {
		return writeErr
	}
	if writeErr := ctx.Reporter.WriteOperations(ctx.Results); writeErr != nil {
		return writeErr
	}
	summary := FormatInstallSummary(result, nil)
	if writeErr := ctx.Reporter.WriteText("summary", summary); writeErr != nil {
		return writeErr
	}
	if code != ExitInstallAdminRequired && !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func detectionOperationStatus(report detector.DetectionReport) app.OperationStatus {
	if report.Health == detector.GrabberHealthHealthy || report.Health == detector.GrabberHealthNotInstalled {
		return app.OperationStatusSuccess
	}
	return app.OperationStatusWarning
}

func msiOperationStatus(result MSIResult) app.OperationStatus {
	if !result.Success {
		return app.OperationStatusFailed
	}
	if result.RebootRequired {
		return app.OperationStatusWarning
	}
	return app.OperationStatusSuccess
}

func verificationOperationStatus(result VerificationResult) app.OperationStatus {
	switch result.Status {
	case VerificationSuccess:
		return app.OperationStatusSuccess
	case VerificationWarning:
		return app.OperationStatusWarning
	default:
		return app.OperationStatusFailed
	}
}

func installExitCode(msi MSIResult, verification VerificationResult) int {
	if !msi.Success {
		return ExitInstallMSIFailed
	}
	if verification.Status == VerificationFailed {
		return ExitInstallVerificationFailed
	}
	if msi.RebootRequired {
		return ExitInstallRebootRequired
	}
	if verification.Status == VerificationWarning {
		return ExitInstallWarnings
	}
	return ExitInstallSuccess
}
