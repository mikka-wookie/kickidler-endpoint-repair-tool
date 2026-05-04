package recommendations

import "strings"

const defaultInstallerPlaceholder = `.\grabberEM.x64.msi`

func RepairCommand(installerPath string) string {
	return `.\kigrepair.exe repair --installer "` + installerForCommand(installerPath) + `" --invite "<INVITE>" --yes`
}

func CleanupDryRunCommand() string {
	return `.\kigrepair.exe cleanup --dry-run`
}

func CleanupThenRepairCommand(installerPath string) string {
	return `.\kigrepair.exe cleanup --yes` + "\n" + RepairCommand(installerPath)
}

func DefenderEnsureCommand() string {
	return `.\kigrepair.exe defender --ensure --yes`
}

func CollectReportCommand() string {
	return `.\kigrepair.exe collect-report`
}

func installerForCommand(installerPath string) string {
	installerPath = strings.TrimSpace(installerPath)
	if installerPath == "" {
		return defaultInstallerPlaceholder
	}
	return strings.ReplaceAll(installerPath, `"`, ``)
}
