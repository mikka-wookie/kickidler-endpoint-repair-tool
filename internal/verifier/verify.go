package verifier

import (
	"strings"
	"time"

	"kigrepair/internal/detector"
)

func VerifyInstallation(report detector.DetectionReport, opts VerifyOptions) VerificationResult {
	started := time.Now()
	if !opts.ExpectInstalledState {
		opts.ExpectInstalledState = true
	}
	result := VerificationResult{
		StartedAt:         started,
		Health:            string(report.Health),
		InstallMode:       string(report.InstallMode),
		InstallRoot:       report.InstallRoot,
		PrimaryService:    report.PrimaryService,
		ServiceStatus:     serviceStatus(report),
		ServiceExecutable: report.ServiceExecutablePath,
		Detection:         report,
	}

	addFinalHealthCheck(&result, report)
	addInstallRootCheck(&result, report, opts)
	addInstallModeCheck(&result, report, opts)
	addPrimaryServiceCheck(&result, report)
	addServiceRunningCheck(&result, report, opts)
	addImagePathCheck(&result, report)
	addExecutableCheck(&result, report)
	if opts.RequireRunningProcess {
		addProcessCheck(&result, report)
	}
	addDefenderCheck(&result, report, opts)
	if opts.InstallExecuted {
		addMSILogCheck(&result, opts.MSIInstallLogPath)
	}

	result.OverallStatus = overallStatus(result.Checks)
	result.FinishedAt = time.Now()
	return result
}

func addFinalHealthCheck(result *VerificationResult, report detector.DetectionReport) {
	check := VerificationCheck{Name: "final_health", Target: "grabber", Status: VerificationPassed, Message: "Final health is healthy"}
	if report.Health != detector.GrabberHealthHealthy {
		check.Status = VerificationFailed
		check.Message = "Final health is " + string(report.Health)
		check.Error = check.Message
	}
	addCheck(result, check)
}

func addInstallRootCheck(result *VerificationResult, report detector.DetectionReport, opts VerifyOptions) {
	check := VerificationCheck{Name: "install_root_detected", Target: report.InstallRoot, Status: VerificationPassed, Message: "Install root detected"}
	if strings.TrimSpace(report.InstallRoot) == "" && opts.ExpectInstalledState {
		check.Status = VerificationFailed
		check.Message = "Install root was not detected"
		check.Error = check.Message
	}
	addCheck(result, check)
}

func addInstallModeCheck(result *VerificationResult, report detector.DetectionReport, opts VerifyOptions) {
	check := VerificationCheck{Name: "install_mode_detected", Target: string(report.InstallMode), Status: VerificationPassed, Message: "Install mode detected"}
	if report.InstallMode == detector.InstallModeNotInstalled && opts.ExpectInstalledState {
		check.Status = VerificationFailed
		check.Message = "Install mode is not_installed"
		check.Error = check.Message
	} else if !installModeDetected(report.InstallMode) {
		check.Status = VerificationWarning
		check.Message = "Install mode is " + string(report.InstallMode)
	}
	addCheck(result, check)
}

func addPrimaryServiceCheck(result *VerificationResult, report detector.DetectionReport) {
	service := primaryService(report)
	check := VerificationCheck{Name: "primary_service_exists", Target: report.PrimaryService, Status: VerificationPassed, Message: "Primary service exists"}
	if service == nil {
		check.Status = VerificationFailed
		check.Message = "Primary service was not found"
		check.Error = check.Message
	}
	addCheck(result, check)
}

func addServiceRunningCheck(result *VerificationResult, report detector.DetectionReport, opts VerifyOptions) {
	check := VerificationCheck{Name: "primary_service_running", Target: report.PrimaryService, Status: VerificationPassed, Message: "Primary service is running"}
	if serviceRunning(report) {
		addCheck(result, check)
		return
	}
	if primaryService(report) != nil && serviceExecutableExists(report) && opts.AllowStoppedServiceWarning {
		check.Status = VerificationWarning
		check.Message = "Service exists and executable exists, but service is not running"
		addCheck(result, check)
		return
	}
	check.Status = VerificationFailed
	check.Message = "Primary service is not running"
	check.Error = check.Message
	addCheck(result, check)
}

func addImagePathCheck(result *VerificationResult, report detector.DetectionReport) {
	check := VerificationCheck{Name: "service_image_path_usable", Target: report.PrimaryServiceImagePath, Status: VerificationPassed, Message: "Service ImagePath is usable"}
	if !serviceImagePathUsable(report) {
		check.Status = VerificationFailed
		check.Message = "Service ImagePath could not be parsed"
		check.Error = check.Message
	}
	addCheck(result, check)
}

func addExecutableCheck(result *VerificationResult, report detector.DetectionReport) {
	check := VerificationCheck{Name: "service_executable_exists", Target: report.ServiceExecutablePath, Status: VerificationPassed, Message: "Service executable exists"}
	if !serviceExecutableExists(report) {
		check.Status = VerificationFailed
		check.Message = "Service executable is missing"
		check.Error = check.Message
	}
	addCheck(result, check)
}

func addProcessCheck(result *VerificationResult, report detector.DetectionReport) {
	found, name := expectedProcessRunning(report)
	check := VerificationCheck{Name: "expected_process_running", Target: report.InstallRoot, Status: VerificationPassed}
	if found {
		check.Message = name + " running"
		result.ProcessStatus = "found"
		addCheck(result, check)
		return
	}
	result.ProcessStatus = "warning"
	check.Status = VerificationWarning
	if serviceRunning(report) {
		check.Message = "No matching Grabber process was detected, but service is running"
	} else {
		check.Message = "No matching Grabber process was detected"
	}
	addCheck(result, check)
}

func addDefenderCheck(result *VerificationResult, report detector.DetectionReport, opts VerifyOptions) {
	check := VerificationCheck{Name: "defender_exclusion_covered", Target: strings.Join(report.RequiredDefenderPaths, "; "), Status: VerificationPassed, Message: "Defender exclusion covers detected install root"}
	if defenderCovered(report) {
		result.DefenderStatus = "present"
		addCheck(result, check)
		return
	}
	check.Status = VerificationWarning
	if !report.Defender.Available {
		result.DefenderStatus = "unavailable"
		check.Message = "Defender exclusions could not be verified"
		if !opts.AllowDefenderUnavailable {
			check.Status = VerificationFailed
			check.Error = check.Message
		}
	} else {
		result.DefenderStatus = "warning"
		check.Message = "Defender exclusion is missing for detected install root"
	}
	addCheck(result, check)
}

func addMSILogCheck(result *VerificationResult, path string) {
	check := VerificationCheck{Name: "msi_install_log_exists", Target: path, Status: VerificationPassed, Message: "MSI install log exists"}
	if !msiInstallLogExists(path) {
		check.Status = VerificationWarning
		check.Message = "MSI install log is missing"
		result.MSIInstallLogState = "missing"
	} else {
		result.MSIInstallLogState = "present"
	}
	addCheck(result, check)
}

func addCheck(result *VerificationResult, check VerificationCheck) {
	result.Checks = append(result.Checks, check)
	if check.Status == VerificationWarning && check.Message != "" {
		result.Warnings = append(result.Warnings, check.Message)
	}
	if check.Status == VerificationFailed {
		if check.Error != "" {
			result.Errors = append(result.Errors, check.Error)
		} else if check.Message != "" {
			result.Errors = append(result.Errors, check.Message)
		}
	}
}

func overallStatus(checks []VerificationCheck) VerificationStatus {
	status := VerificationPassed
	for _, check := range checks {
		switch check.Status {
		case VerificationFailed:
			return VerificationFailed
		case VerificationWarning:
			status = VerificationWarning
		}
	}
	return status
}
