package repair

import (
	"fmt"
	"path/filepath"
	"strings"

	"kigrepair/internal/app"
	"kigrepair/internal/installer"
)

func (w RepairWorkflow) runInstall(ctx *app.AppContext, installerPath string, invite string) installer.MSIResult {
	msiLogPath := filepath.Join(ctx.OutputDir, "msi-install.log")
	executor := w.MSIExecutor
	if executor == nil {
		executor = installer.ExecMSIExecutor{}
	}
	ctx.Logger.Info("msiexec started")
	ctx.Logger.Info("executing command: msiexec.exe %s", strings.Join(installer.MaskedMSIInstallArgs(installerPath, msiLogPath), " "))
	commandResult := executor.Install(installerPath, invite, msiLogPath)
	msi := installer.ClassifyMSIInstallExitCode(commandResult.ExitCode)
	ctx.Logger.Info("MSI exit code: %d", msi.ExitCode)
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
