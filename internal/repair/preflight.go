package repair

import (
	"fmt"
	"strings"
	"time"

	"kigrepair/internal/app"
)

type PreflightWorkflow struct {
	Invite    string `json:"-"`
	Installer string `json:"installer,omitempty"`
}

func (w PreflightWorkflow) Name() string {
	return "preflight"
}

func (w PreflightWorkflow) Run(ctx *app.AppContext) error {
	workflow := DryRunWorkflow{Invite: w.Invite, Installer: w.Installer}
	result := workflow.runPreflight(ctx)
	ctx.JSONValue = result
	ctx.ExitCode = app.ExitSuccess
	if result.HasFailedRequiredCheck() {
		ctx.ExitCode = app.ExitVerificationFailed
	}
	if err := ctx.Reporter.WriteJSON("preflight-result", result); err != nil {
		return err
	}
	if result.InstallerValidation != nil {
		if err := ctx.Reporter.WriteJSON("installer-validation", result.InstallerValidation); err != nil {
			return err
		}
	}
	addOperation(ctx, "preflight", "repair", preflightOperationStatus(result), "Preflight checks completed", strings.Join(result.Errors, "; "))
	if err := ctx.Reporter.WriteOperations(ctx.Results); err != nil {
		return err
	}
	summary := FormatPreflightSummary(result, ctx.ExitCode)
	if err := ctx.Reporter.WriteText("summary", summary); err != nil {
		return err
	}
	if !ctx.Quiet && !ctx.JSONOutput {
		fmt.Print(summary)
	}
	return nil
}

func FormatPreflightSummary(result PreflightResult, exitCode int) string {
	var b strings.Builder
	b.WriteString("Kigrepair Preflight\n\n")
	for _, check := range result.Checks {
		b.WriteString(check.Name + ": " + check.Status + "\n")
		if check.Status == "fail" && check.Error != "" {
			b.WriteString("  " + check.Error + "\n")
		}
	}
	if result.InstallerValidation != nil {
		b.WriteString("\nInstaller validation: " + result.InstallerValidation.Status + "\n")
		b.WriteString("Installer: " + result.InstallerValidation.Path + "\n")
		if result.InstallerValidation.SHA256 != "" {
			b.WriteString("SHA-256: " + result.InstallerValidation.SHA256 + "\n")
		}
		if len(result.InstallerValidation.Warnings) > 0 {
			b.WriteString(fmt.Sprintf("Warnings: %d\n", len(result.InstallerValidation.Warnings)))
		}
	}
	b.WriteString("\nReport:\n")
	b.WriteString(result.ReportDir)
	b.WriteString("\n")
	_ = time.Now()
	_ = exitCode
	return b.String()
}
