package app

import "testing"

func TestStandardExitCodes(t *testing.T) {
	tests := map[string]int{
		"success":               ExitSuccess,
		"warnings":              ExitWarnings,
		"admin required":        ExitAdminRequired,
		"confirmation required": ExitConfirmationRequired,
		"invalid input":         ExitInvalidInput,
		"cleanup failed":        ExitCleanupFailed,
		"install failed":        ExitInstallFailed,
		"verification failed":   ExitVerificationFailed,
		"defender failed":       ExitDefenderFailed,
		"reboot required":       ExitRebootRequired,
		"unexpected error":      ExitUnexpectedError,
	}
	want := map[string]int{
		"success":               0,
		"warnings":              1,
		"admin required":        2,
		"confirmation required": 3,
		"invalid input":         7,
		"cleanup failed":        7,
		"install failed":        7,
		"verification failed":   7,
		"defender failed":       7,
		"reboot required":       9,
		"unexpected error":      10,
	}
	for name, got := range tests {
		if got != want[name] {
			t.Fatalf("%s exit = %d, want %d", name, got, want[name])
		}
	}
}
