package repair

import (
	"strings"

	"kigrepair/internal/detector"
)

type Verification struct {
	Status   string   `json:"status"`
	Message  string   `json:"message"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

func VerifyFinalState(report detector.DetectionReport) Verification {
	serviceExists, serviceRunning := primaryServiceState(report)
	filesHealthy := strings.TrimSpace(report.ServiceExecutablePath) != "" && report.ServiceExecutableExists
	if report.Health != detector.GrabberHealthHealthy {
		return Verification{Status: "failed", Message: "Final verification failed", Errors: []string{"Final health is " + string(report.Health)}}
	}
	if !serviceExists {
		return Verification{Status: "failed", Message: "Final verification failed", Errors: []string{"Primary service was not found"}}
	}
	if !serviceRunning {
		return Verification{Status: "failed", Message: "Final verification failed", Errors: []string{"Primary service is not running"}}
	}
	if !filesHealthy {
		return Verification{Status: "failed", Message: "Final verification failed", Errors: []string{"Service executable is missing"}}
	}
	if strings.TrimSpace(report.InstallRoot) == "" {
		return Verification{Status: "failed", Message: "Final verification failed", Errors: []string{"Install root was not detected"}}
	}

	var warnings []string
	if !report.Defender.Available {
		warnings = append(warnings, "Defender exclusions could not be verified")
	} else if len(report.MissingDefenderPaths) > 0 {
		for _, path := range report.MissingDefenderPaths {
			warnings = append(warnings, "Missing Defender exclusion: "+path)
		}
	}
	if len(warnings) > 0 {
		return Verification{Status: "warning", Message: "Final verification completed with warnings", Warnings: warnings}
	}
	return Verification{Status: "success", Message: "Final verification succeeded"}
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
