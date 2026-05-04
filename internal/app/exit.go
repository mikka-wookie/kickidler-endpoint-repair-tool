package app

const (
	ExitSuccess              = 0
	ExitWarnings             = 1
	ExitAdminRequired        = 2
	ExitConfirmationRequired = 3
	ExitInvalidInput         = 4
	ExitCleanupFailed        = 5
	ExitInstallFailed        = 6
	ExitVerificationFailed   = 7
	ExitDefenderFailed       = 8
	ExitRebootRequired       = 9
	ExitUnexpectedError      = 10
)

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return ""
}
