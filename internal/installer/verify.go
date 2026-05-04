package installer

import (
	"kigrepair/internal/detector"
	"kigrepair/internal/verifier"
)

type VerificationStatus string

const (
	VerificationSuccess VerificationStatus = "success"
	VerificationWarning VerificationStatus = "warning"
	VerificationFailed  VerificationStatus = "failed"
)

type VerificationResult struct {
	Status   VerificationStatus `json:"status"`
	Warnings []string           `json:"warnings,omitempty"`
	Errors   []string           `json:"errors,omitempty"`
	Message  string             `json:"message"`
}

func VerifyInstall(msi MSIResult, final detector.DetectionReport) VerificationResult {
	if !msi.Success {
		return VerificationResult{
			Status:  VerificationFailed,
			Errors:  []string{msi.Message},
			Message: "MSI install failed",
		}
	}
	result := verifier.VerifyInstallation(final, verifier.VerifyOptions{
		AllowDefenderUnavailable:   true,
		ExpectInstalledState:       true,
		AllowStoppedServiceWarning: true,
	})
	switch result.OverallStatus {
	case verifier.VerificationPassed:
		return VerificationResult{Status: VerificationSuccess, Message: "Install completed successfully"}
	case verifier.VerificationWarning:
		return VerificationResult{Status: VerificationWarning, Warnings: result.Warnings, Message: "Install completed with warning"}
	default:
		return VerificationResult{Status: VerificationFailed, Errors: result.Errors, Message: "Post-install verification failed"}
	}
}
