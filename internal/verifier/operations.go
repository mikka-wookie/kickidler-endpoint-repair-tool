package verifier

import (
	"time"

	"kigrepair/internal/app"
)

func Operations(result VerificationResult) []app.OperationResult {
	operations := []app.OperationResult{{
		Step:      "final_verification",
		Target:    "grabber",
		Status:    operationStatus(result.OverallStatus),
		Message:   "Final verification result: " + string(result.OverallStatus),
		Error:     joinErrors(result.Errors),
		Timestamp: time.Now(),
	}}
	for _, check := range result.Checks {
		step := operationStep(check.Name)
		if step == "" {
			continue
		}
		operations = append(operations, app.OperationResult{
			Step:      step,
			Target:    check.Target,
			Status:    operationStatus(check.Status),
			Message:   check.Message,
			Error:     check.Error,
			Timestamp: time.Now(),
		})
	}
	return operations
}

func operationStep(checkName string) string {
	switch checkName {
	case "primary_service_exists", "primary_service_running", "service_image_path_usable":
		return "verify_service"
	case "service_executable_exists":
		return "verify_executable"
	case "expected_process_running":
		return "verify_process"
	case "defender_exclusion_covered":
		return "verify_defender"
	case "msi_install_log_exists":
		return "verify_msi_log"
	default:
		return ""
	}
}

func operationStatus(status VerificationStatus) app.OperationStatus {
	switch status {
	case VerificationPassed:
		return app.OperationStatusSuccess
	case VerificationWarning:
		return app.OperationStatusWarning
	case VerificationSkipped:
		return app.OperationStatusSkipped
	default:
		return app.OperationStatusFailed
	}
}

func joinErrors(values []string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		result += "; " + value
	}
	return result
}
