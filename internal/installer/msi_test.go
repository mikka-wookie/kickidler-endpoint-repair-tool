package installer

import "testing"

func TestClassifyMSIInstallExitCode(t *testing.T) {
	tests := []struct {
		code           int
		status         string
		success        bool
		rebootRequired bool
	}{
		{code: 0, status: "success", success: true},
		{code: -1, status: "failed_timeout"},
		{code: 3010, status: "success_reboot_required", success: true, rebootRequired: true},
		{code: 1603, status: "failed_fatal_error"},
		{code: 1619, status: "failed_package_open"},
		{code: 1620, status: "failed_invalid_package"},
		{code: 1633, status: "failed_platform_unsupported"},
		{code: 9999, status: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := ClassifyMSIInstallExitCode(tt.code)
			if got.Status != tt.status {
				t.Fatalf("status = %q, want %q", got.Status, tt.status)
			}
			if got.Success != tt.success {
				t.Fatalf("success = %t, want %t", got.Success, tt.success)
			}
			if got.RebootRequired != tt.rebootRequired {
				t.Fatalf("rebootRequired = %t, want %t", got.RebootRequired, tt.rebootRequired)
			}
		})
	}
}
