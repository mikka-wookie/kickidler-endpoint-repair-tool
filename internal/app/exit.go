package app

const (
	ExitSuccess              = 0
	ExitWarnings             = 1
	ExitAdminRequired        = 2
	ExitConfirmationRequired = 3
	ExitInvalidInput         = 7
	ExitCleanupFailed        = 7
	ExitInstallFailed        = 7
	ExitVerificationFailed   = 7
	ExitDefenderFailed       = 7
	ExitRebootRequired       = 9
	ExitUnexpectedError      = 10
)

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return ""
}
