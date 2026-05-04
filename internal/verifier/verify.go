package verifier

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"kigrepair/internal/detector"
)

func Verify(report detector.DetectionReport, opts Options) VerificationResult {
	result := VerificationResult{
		Status:          VerificationSuccess,
		Message:         "Verification succeeded",
		Detection:       report,
		InstallExecuted: opts.InstallExecuted,
		MSIInstallLog:   opts.MSIInstallLog,
	}

	result.add("final_health", string(report.Health), statusForHealth(report.Health), "Final health is "+string(report.Health))
	result.add("install_root_detected", report.InstallRoot, requiredStringStatus(report.InstallRoot), requiredStringMessage("Install root", report.InstallRoot))
	result.add("install_mode_detected", string(report.InstallMode), installModeStatus(report.InstallMode), "Install mode is "+string(report.InstallMode))

	serviceExists, serviceRunning := primaryServiceState(report)
	result.add("primary_service_exists", report.PrimaryService, boolStatus(serviceExists), boolMessage("Primary service exists", serviceExists))
	result.add("primary_service_running", report.PrimaryService, boolStatus(serviceRunning), boolMessage("Primary service is running", serviceRunning))

	executableTarget := report.ServiceExecutablePath
	result.add("service_executable_exists", executableTarget, boolStatus(strings.TrimSpace(executableTarget) != "" && report.ServiceExecutableExists), boolMessage("Service executable exists", strings.TrimSpace(executableTarget) != "" && report.ServiceExecutableExists))

	processStatus, processTarget, processMessage := processCheck(report)
	result.add("expected_grabber_process_running", processTarget, processStatus, processMessage)

	defenderStatus, defenderTarget, defenderMessage := defenderCheck(report)
	result.add("defender_exclusion_coverage", defenderTarget, defenderStatus, defenderMessage)

	msiStatus, msiTarget, msiMessage := msiLogCheck(opts)
	result.add("msi_install_log_exists", msiTarget, msiStatus, msiMessage)

	result.classify()
	return result
}

func (r *VerificationResult) add(name string, target string, status CheckStatus, message string) {
	r.Checks = append(r.Checks, VerificationCheck{
		Name:      name,
		Target:    target,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	})
}

func (r *VerificationResult) classify() {
	for _, check := range r.Checks {
		switch check.Status {
		case CheckFailed:
			r.Errors = append(r.Errors, check.Message)
		case CheckWarning:
			r.Warnings = append(r.Warnings, check.Message)
		}
	}
	if len(r.Errors) > 0 {
		r.Status = VerificationFailed
		r.Message = "Verification failed"
		return
	}
	if len(r.Warnings) > 0 {
		r.Status = VerificationWarning
		r.Message = "Verification completed with warnings"
		return
	}
	r.Status = VerificationSuccess
	r.Message = "Verification succeeded"
}

func statusForHealth(health detector.GrabberHealthStatus) CheckStatus {
	if health == detector.GrabberHealthHealthy {
		return CheckSuccess
	}
	return CheckFailed
}

func requiredStringStatus(value string) CheckStatus {
	if strings.TrimSpace(value) == "" {
		return CheckFailed
	}
	return CheckSuccess
}

func requiredStringMessage(name string, value string) string {
	if strings.TrimSpace(value) == "" {
		return name + " was not detected"
	}
	return name + " detected"
}

func installModeStatus(mode detector.InstallMode) CheckStatus {
	if mode == detector.InstallModeUnknown || mode == detector.InstallModeNotInstalled || strings.TrimSpace(string(mode)) == "" {
		return CheckFailed
	}
	return CheckSuccess
}

func boolStatus(value bool) CheckStatus {
	if value {
		return CheckSuccess
	}
	return CheckFailed
}

func boolMessage(message string, value bool) string {
	if value {
		return message
	}
	return message + ": no"
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

func processCheck(report detector.DetectionReport) (CheckStatus, string, string) {
	if len(report.Processes) == 0 {
		return CheckWarning, "", "Expected Grabber process was not detected"
	}
	serviceExecutable := normalizeOptionalPath(report.ServiceExecutablePath)
	installRoot := normalizeOptionalPath(report.InstallRoot)
	for _, process := range report.Processes {
		processPath := normalizeOptionalPath(process.ExecutablePath)
		target := process.Name
		if process.ExecutablePath != "" {
			target = process.ExecutablePath
		}
		if serviceExecutable != "" && strings.EqualFold(processPath, serviceExecutable) {
			return CheckSuccess, target, "Expected Grabber process is running"
		}
		if installRoot != "" && detector.IsPathCoveredByExclusion(processPath, installRoot) {
			return CheckSuccess, target, "Expected Grabber process is running"
		}
		if process.MatchedByExactPath {
			return CheckSuccess, target, "Expected Grabber process is running"
		}
	}
	return CheckWarning, "", "Grabber process query succeeded, but no process matched the detected install root"
}

func normalizeOptionalPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return detector.NormalizeWindowsPath(path)
}

func defenderCheck(report detector.DetectionReport) (CheckStatus, string, string) {
	target := strings.Join(report.RequiredDefenderPaths, "; ")
	if !report.Defender.Available {
		return CheckWarning, target, "Defender exclusions could not be verified"
	}
	if len(report.MissingDefenderPaths) > 0 {
		return CheckWarning, strings.Join(report.MissingDefenderPaths, "; "), "Missing Defender exclusion coverage"
	}
	if len(report.RequiredDefenderPaths) == 0 {
		return CheckSkipped, "", "No Defender exclusion path is required for the detected state"
	}
	return CheckSuccess, target, "Defender exclusion coverage is present"
}

func msiLogCheck(opts Options) (CheckStatus, string, string) {
	if !opts.InstallExecuted {
		return CheckSkipped, "", "MSI install was not executed in this verification scope"
	}
	if strings.TrimSpace(opts.MSIInstallLog) == "" {
		return CheckFailed, "", "MSI install log path was not provided"
	}
	cleanPath := filepath.Clean(opts.MSIInstallLog)
	if _, err := os.Stat(cleanPath); err != nil {
		return CheckFailed, cleanPath, "MSI install log does not exist"
	}
	return CheckSuccess, cleanPath, "MSI install log exists"
}
