package verifier

import (
	"os"
	"path/filepath"
	"strings"

	"kigrepair/internal/detector"
)

func serviceByName(report detector.DetectionReport, name string) *detector.ServiceState {
	for i := range report.Services {
		if strings.EqualFold(report.Services[i].Name, name) {
			return &report.Services[i]
		}
	}
	return nil
}

func primaryService(report detector.DetectionReport) *detector.ServiceState {
	if strings.TrimSpace(report.PrimaryService) != "" {
		if service := serviceByName(report, report.PrimaryService); service != nil && service.Exists {
			return service
		}
	}
	for i := range report.Services {
		if report.Services[i].Exists {
			return &report.Services[i]
		}
	}
	return nil
}

func serviceStatus(report detector.DetectionReport) string {
	service := primaryService(report)
	if service == nil {
		return ""
	}
	if strings.TrimSpace(service.Status) != "" {
		return service.Status
	}
	if service.Exists {
		return "exists"
	}
	return ""
}

func serviceRunning(report detector.DetectionReport) bool {
	service := primaryService(report)
	return service != nil && strings.EqualFold(service.Status, "running")
}

func serviceImagePathUsable(report detector.DetectionReport) bool {
	return strings.TrimSpace(report.PrimaryServiceImagePath) != "" && strings.TrimSpace(report.ServiceExecutablePath) != ""
}

func serviceExecutableExists(report detector.DetectionReport) bool {
	if strings.TrimSpace(report.ServiceExecutablePath) == "" {
		return false
	}
	if report.ServiceExecutableExists {
		return true
	}
	if _, err := os.Stat(report.ServiceExecutablePath); err == nil {
		return true
	}
	return false
}

func installModeDetected(mode detector.InstallMode) bool {
	switch mode {
	case detector.InstallModeStandard, detector.InstallModeHelper, detector.InstallModeHiddenWMI:
		return true
	default:
		return false
	}
}

func expectedProcessRunning(report detector.DetectionReport) (bool, string) {
	root := report.InstallRoot
	if report.InstallMode == detector.InstallModeHiddenWMI && report.BinaryDir != "" {
		root = report.BinaryDir
	}
	for _, process := range report.Processes {
		if !detector.TrustedProcessForTermination(process) || process.ExecutablePath == "" {
			continue
		}
		if report.InstallMode == detector.InstallModeHiddenWMI {
			if process.TrustLevel == detector.ProcessTrustHiddenWMIExactPath && pathInsideOrEqual(process.ExecutablePath, root) {
				return true, process.Name
			}
			continue
		}
		if process.TrustLevel == detector.ProcessTrustNameAndPathMatch && pathInsideOrEqual(process.ExecutablePath, root) {
			return true, process.Name
		}
	}
	return false, ""
}

func pathInsideOrEqual(path string, root string) bool {
	path = strings.ToLower(detector.NormalizeWindowsPath(path))
	root = strings.ToLower(detector.NormalizeWindowsPath(root))
	if path == "" || root == "" || path == "." || root == "." {
		return false
	}
	if path == root {
		return true
	}
	if strings.HasPrefix(path, root+`\`) {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

func defenderCovered(report detector.DetectionReport) bool {
	return report.Defender.Available && len(report.MissingDefenderPaths) == 0 && len(report.RequiredDefenderPaths) > 0
}

func msiInstallLogExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
