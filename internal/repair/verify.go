package repair

import (
	"kigrepair/internal/detector"
	"kigrepair/internal/verifier"
)

type Verification struct {
	Status   string   `json:"status"`
	Message  string   `json:"message"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func VerifyFinalState(report detector.DetectionReport) Verification {
	result := verifier.VerifyInstallation(report, verifier.VerifyOptions{
		AllowDefenderUnavailable:   true,
		ExpectInstalledState:       true,
		AllowStoppedServiceWarning: false,
	})
	switch result.OverallStatus {
	case verifier.VerificationPassed:
		return Verification{Status: "success", Message: "Final verification succeeded"}
	case verifier.VerificationWarning:
		return Verification{Status: "warning", Message: "Final verification completed with warnings", Warnings: result.Warnings}
	default:
		return Verification{Status: "failed", Message: "Final verification failed", Errors: result.Errors}
	}
}
