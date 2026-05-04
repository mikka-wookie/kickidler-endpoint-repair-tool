package repair

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/installer"
)

func msiInstallLogPath(outputDir string) string {
	return filepath.Join(outputDir, "msi-install.log")
}

func (w RepairWorkflow) runInstall(ctx *app.AppContext, installerPath string, invite string) installer.MSIResult {
	msiLogPath := msiInstallLogPath(ctx.OutputDir)
	executor := w.MSIExecutor
	if executor == nil {
		executor = installer.ExecMSIExecutor{}
	}
	ctx.Logger.Info("msiexec started")
	ctx.Logger.Info("executing command: msiexec.exe %s", strings.Join(installer.MaskedMSIInstallArgs(installerPath, msiLogPath), " "))
	msiStarted := time.Now()
	commandResult := executor.Install(installerPath, invite, msiLogPath)
	msi := installer.ClassifyMSIInstallExitCode(commandResult.ExitCode)
	ctx.Logger.Info("MSI exit code: %d", msi.ExitCode)
	ctx.Logger.Info("MSI duration: %s", time.Since(msiStarted).Round(time.Millisecond))
	if strings.TrimSpace(commandResult.Output) != "" {
		ctx.Logger.Info("MSI output summary: %s", shortOutput(installer.MaskInviteInText(commandResult.Output, invite)))
	}
	status := app.OperationStatusSuccess
	errText := ""
	if msi.RebootRequired {
		status = app.OperationStatusWarning
	} else if !msi.Success {
		status = app.OperationStatusFailed
		errText = msi.Message
	}
	addOperation(ctx, "msi_install", installerPath, status, fmt.Sprintf("%s; exit code %d", msi.Status, msi.ExitCode), errText)
	return msi
}

func shortOutput(output string) string {
	output = strings.Join(strings.Fields(output), " ")
	if len(output) > 500 {
		return output[:500] + "..."
	}
	return output
}
