package cleaner

import (
	"testing"

	"kigrepair/internal/app"
)

func TestClassifyMSIUninstallExitCode(t *testing.T) {
	tests := []struct {
		code   int
		status app.OperationStatus
	}{
		{0, app.OperationStatusSuccess},
		{1605, app.OperationStatusSkipped},
		{1614, app.OperationStatusSkipped},
		{3010, app.OperationStatusWarning},
		{1234, app.OperationStatusFailed},
	}

	for _, tt := range tests {
		got := ClassifyMSIUninstallExitCode(tt.code)
		if got.Status != tt.status {
			t.Fatalf("code %d status = %s, want %s", tt.code, got.Status, tt.status)
		}
	}
}
