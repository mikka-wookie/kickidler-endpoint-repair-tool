package installer

import (
	"strings"

	"kigrepair/internal/detector"
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

	serviceExists, serviceRunning := primaryServiceState(final)
	filesHealthy := final.ServiceExecutablePath != "" && final.ServiceExecutableExists
	if !serviceExists {
		return VerificationResult{
			Status:  VerificationFailed,
			Errors:  []string{"Known service was not found after install"},
			Message: "Post-install verification failed",
		}
	}
	if !filesHealthy {
		return VerificationResult{
			Status:  VerificationFailed,
			Errors:  []string{"Service executable is missing after install"},
			Message: "Post-install verification failed",
		}
	}
	if final.Health == detector.GrabberHealthBroken || final.Health == detector.GrabberHealthNotInstalled || final.Health == detector.GrabberHealthUnknown {
		return VerificationResult{
			Status:  VerificationFailed,
			Errors:  []string{"Final health is " + string(final.Health)},
			Message: "Post-install verification failed",
		}
	}

	var warnings []string
	if !serviceRunning {
		warnings = append(warnings, "Service exists and executable exists, but service is not running")
	}
	warnings = append(warnings, defenderWarnings(final)...)
	if len(warnings) > 0 {
		return VerificationResult{
			Status:   VerificationWarning,
			Warnings: warnings,
			Message:  "Install completed with warning",
		}
	}
	return VerificationResult{Status: VerificationSuccess, Message: "Install completed successfully"}
}

func primaryServiceState(report detector.DetectionReport) (bool, bool) {
	for _, service := range report.Services {
		if strings.EqualFold(service.Name, report.PrimaryService) {
			return service.Exists, strings.EqualFold(service.Status, "running")
		}
	}
	for _, service := range report.Services {
		if service.Exists {
			return true, strings.EqualFold(service.Status, "running")
		}
	}
	return false, false
}

func defenderWarnings(report detector.DetectionReport) []string {
	var warnings []string
	for _, path := range report.MissingDefenderPaths {
		warnings = append(warnings, "Missing Defender exclusion: "+path)
	}
	if !report.Defender.Available && report.InstallRoot != "" {
		warnings = append(warnings, "Defender exclusions could not be verified")
	}
	return warnings
}
