package failures

const (
	FailureReportWrite      = "report_write_failed"
	FailureExternalCommand  = "external_command_failed"
	FailureExternalTimeout  = "external_command_timeout"
	FailurePowerShell       = "powershell_failed"
	FailureMSI              = "msi_failed"
	FailureServiceControl   = "service_control_failed"
	FailureProcessControl   = "process_control_failed"
	FailureRegistryAccess   = "registry_access_failed"
	FailureDefenderAccess   = "defender_access_failed"
	FailureValidation       = "validation_failed"
	FailurePreflight        = "preflight_failed"
	FailureRollbackSnapshot = "rollback_snapshot_failed"
	FailurePermissionDenied = "permission_denied"
	FailureUnexpected       = "unexpected_error"
)

type WorkflowError struct {
	Code      string `json:"code"`
	Category  string `json:"category"`
	Message   string `json:"message"`
	Cause     string `json:"cause,omitempty"`
	Action    string `json:"action,omitempty"`
	Retryable bool   `json:"retryable"`
	Fatal     bool   `json:"fatal"`
}

func (e WorkflowError) Error() string {
	if e.Cause == "" {
		return e.Message
	}
	return e.Message + ": " + e.Cause
}

func New(code string, category string, message string, cause error, action string, retryable bool, fatal bool) WorkflowError {
	err := WorkflowError{
		Code:      code,
		Category:  category,
		Message:   message,
		Action:    action,
		Retryable: retryable,
		Fatal:     fatal,
	}
	if cause != nil {
		err.Cause = cause.Error()
	}
	return err
}
