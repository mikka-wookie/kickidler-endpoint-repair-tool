package cleaner

import "kigrepair/internal/app"

type MSIExitClassification struct {
	Status  app.OperationStatus
	Message string
}

func ClassifyMSIUninstallExitCode(code int) MSIExitClassification {
	switch code {
	case 0:
		return MSIExitClassification{Status: app.OperationStatusSuccess, Message: "MSI uninstall completed"}
	case 1605:
		return MSIExitClassification{Status: app.OperationStatusSkipped, Message: "MSI product is not installed"}
	case 1614:
		return MSIExitClassification{Status: app.OperationStatusSkipped, Message: "MSI product is already uninstalled"}
	case 3010:
		return MSIExitClassification{Status: app.OperationStatusWarning, Message: "MSI uninstall completed; reboot required"}
	default:
		return MSIExitClassification{Status: app.OperationStatusFailed, Message: "MSI uninstall failed"}
	}
}
