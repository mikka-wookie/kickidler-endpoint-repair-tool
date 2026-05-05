package detector

import (
	"path/filepath"
	"sort"
	"strings"

	svc "kigrepair/internal/services"
)

type InstallMode string

const (
	InstallModeUnknown      InstallMode = "unknown"
	InstallModeStandard     InstallMode = "standard"
	InstallModeHelper       InstallMode = "helper"
	InstallModeHiddenWMI    InstallMode = "hidden_wmi"
	InstallModeNotInstalled InstallMode = "not_installed"
)

var knownServiceNames = []string{"ngs", "tls", wmiProviderService}

var standardExecutableCandidates = []string{
	"grabber.exe",
	"grabberAgent.exe",
	"grabberSubAgent.exe",
	"grabberSubagent.exe",
	"grabber2.exe",
	"ngsAgent.exe",
	"ngsSubAgent.exe",
	"ngsSubagent.exe",
	"tlshost.exe",
	"tlsservice.exe",
	"tlssubservice.exe",
}

var hiddenWMIExecutableCandidates = []string{
	"svchost.exe",
	"WmiPrvSE.exe",
	"RuntimeBroker.exe",
}

type installCandidate struct {
	service    *ServiceState
	exePath    string
	installDir string
	binaryDir  string
	mode       InstallMode
}

func EnrichDetectionReport(report DetectionReport) DetectionReport {
	if report.InstallMode == "" {
		report.InstallMode = InstallModeUnknown
	}
	report.Services = HardenServiceStates(report.System, report.Services)
	candidate := chooseServiceInstallCandidate(report.System, report.Services)
	if candidate == nil && report.InstallRoot == "" {
		candidate = chooseFileInstallCandidate(report.System, report.Files)
	}
	if candidate != nil {
		applyInstallCandidate(&report, candidate)
	}
	if report.InstallRoot == "" && !anyExistingService(report.Services) && !anyExistingFileType(report.Files, "folder") && !anyExistingRegistry(report.Registry) {
		report.InstallMode = InstallModeNotInstalled
	}
	if report.InstallRoot != "" && report.BinaryDir == "" {
		binaryDir := binaryDirForMode(report.InstallMode, report.InstallRoot)
		if !pathsEqual(binaryDir, report.InstallRoot) {
			report.BinaryDir = binaryDir
		}
	}
	if report.ServiceExecutablePath != "" {
		report.ServiceExecutableExists = fileExists(report.Files, report.ServiceExecutablePath)
	}
	report.RequiredDefenderPaths = requiredDefenderPathsForInstall(report.System, report.InstallMode, report.InstallRoot)
	normalizedExclusions, coveredDefenderPaths, missingDefenderPaths := EvaluateDefenderCoverage(report.RequiredDefenderPaths, report.Defender.ExclusionPaths)
	report.MissingDefenderPaths = missingDefenderPaths
	if report.Defender.Available {
		report.Defender.RequiredPaths = append([]string{}, report.RequiredDefenderPaths...)
		report.Defender.NormalizedExclusionPaths = normalizedExclusions
		report.Defender.CoveredPaths = coveredDefenderPaths
		report.Defender.MissingPaths = append([]string{}, missingDefenderPaths...)
	}
	return report
}

func applyInstallCandidate(report *DetectionReport, candidate *installCandidate) {
	report.InstallMode = candidate.mode
	report.InstallRoot = candidate.installDir
	report.BinaryDir = ""
	if candidate.binaryDir != "" && !pathsEqual(candidate.binaryDir, candidate.installDir) {
		report.BinaryDir = candidate.binaryDir
	}
	report.ServiceExecutablePath = candidate.exePath
	if candidate.service != nil {
		report.ServiceSource = candidate.service.Name
		report.PrimaryService = candidate.service.Name
		report.PrimaryServiceImagePath = candidate.service.ImagePath
	}
}

func chooseServiceInstallCandidate(system SystemState, services []ServiceState) *installCandidate {
	candidates := make([]installCandidate, 0, len(services))
	for i := range services {
		service := &services[i]
		if !service.Exists || service.TrustLevel != svc.TrustTrusted {
			continue
		}
		exePath := service.NormalizedExecutablePath
		if exePath == "" {
			exePath = parseServiceExecutablePath(service.ImagePath)
		}
		if exePath == "" || service.InstallRoot == "" {
			continue
		}
		root := service.InstallRoot
		mode := InstallMode(service.InstallMode)
		if mode == "" {
			mode = InstallModeUnknown
		}
		binaryDir := binaryDirForMode(mode, root)
		if root == "" {
			continue
		}
		candidates = append(candidates, installCandidate{
			service:    service,
			exePath:    exePath,
			installDir: root,
			binaryDir:  binaryDir,
			mode:       mode,
		})
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return serviceCandidatePriority(system, candidates[i]) < serviceCandidatePriority(system, candidates[j])
	})
	return &candidates[0]
}

func serviceCandidatePriority(system SystemState, candidate installCandidate) int {
	if candidate.service != nil && strings.EqualFold(candidate.service.Name, wmiProviderService) && isHiddenWMIExecutable(system, candidate.exePath) {
		return 1
	}
	if candidate.service != nil && strings.EqualFold(candidate.service.Name, "tls") {
		return 2
	}
	if candidate.service != nil && strings.EqualFold(candidate.service.Name, "ngs") {
		return 3
	}
	return 4
}

func chooseFileInstallCandidate(system SystemState, files []FileState) *installCandidate {
	roots := knownInstallRoots(system)
	for _, root := range roots {
		mode := modeForInstallRoot(system, root)
		binaryDir := binaryDirForMode(mode, root)
		for _, exe := range executableCandidatesForMode(mode) {
			path := filepath.Join(binaryDir, exe)
			if fileExists(files, path) {
				return &installCandidate{
					exePath:    path,
					installDir: root,
					binaryDir:  binaryDir,
					mode:       mode,
				}
			}
		}
	}
	return nil
}

func parseServiceExecutablePath(imagePath string) string {
	parsed := svc.ParseServiceImagePath(imagePath)
	return parsed.NormalizedPath
}

func classifyExecutablePath(system SystemState, exePath string) (string, string, InstallMode) {
	clean := NormalizeWindowsPath(exePath)
	if isHiddenWMIExecutable(system, clean) {
		return hiddenWMIRoot(system), hiddenWMIBinaryDir(system), InstallModeHiddenWMI
	}
	for _, root := range []string{
		filepath.Join(system.ProgramFiles, "TeleLinkSoft"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoft"),
	} {
		if pathWithin(clean, root) {
			return root, filepath.Join(root, "bin"), InstallModeStandard
		}
	}
	for _, root := range []string{
		filepath.Join(system.ProgramFiles, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoftHelper"),
	} {
		if pathWithin(clean, root) {
			return root, root, InstallModeHelper
		}
	}
	return "", "", InstallModeUnknown
}

func isHiddenWMIExecutable(system SystemState, exePath string) bool {
	dir := filepath.Dir(NormalizeWindowsPath(exePath))
	if !pathsEqual(dir, hiddenWMIBinaryDir(system)) {
		return false
	}
	for _, name := range hiddenWMIExecutableCandidates {
		if strings.EqualFold(filepath.Base(exePath), name) {
			return true
		}
	}
	return false
}

func pathWithin(child, parent string) bool {
	child = strings.ToLower(NormalizeWindowsPath(child))
	parent = strings.ToLower(NormalizeWindowsPath(parent))
	if child == "" || parent == "" {
		return false
	}
	if child == parent {
		return true
	}
	return strings.HasPrefix(child, strings.TrimRight(parent, `\`)+`\`)
}

func knownInstallRoots(system SystemState) []string {
	return []string{
		filepath.Join(system.ProgramFiles, "TeleLinkSoft"),
		filepath.Join(system.ProgramFiles, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoft"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramData, "E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60"),
		hiddenWMIRoot(system),
		hiddenWMIBinaryDir(system),
	}
}

func modeForInstallRoot(system SystemState, root string) InstallMode {
	switch {
	case pathsEqual(root, hiddenWMIRoot(system)) || pathsEqual(root, hiddenWMIBinaryDir(system)):
		return InstallModeHiddenWMI
	case strings.Contains(strings.ToLower(root), strings.ToLower("TeleLinkSoftHelper")):
		return InstallModeHelper
	case strings.Contains(strings.ToLower(root), strings.ToLower("TeleLinkSoft")):
		return InstallModeStandard
	default:
		return InstallModeUnknown
	}
}

func binaryDirForMode(mode InstallMode, root string) string {
	if mode == InstallModeHiddenWMI && !strings.EqualFold(filepath.Base(root), "bin") {
		return filepath.Join(root, "bin")
	}
	return root
}

func executableCandidatesForMode(mode InstallMode) []string {
	if mode == InstallModeHiddenWMI {
		return hiddenWMIExecutableCandidates
	}
	return standardExecutableCandidates
}

func requiredFilePathsForInstall(report DetectionReport) []string {
	if report.ServiceExecutablePath != "" {
		return []string{report.ServiceExecutablePath}
	}
	if report.InstallMode == InstallModeHiddenWMI {
		return requiredWMIFilePaths(report.System)
	}
	return nil
}

func requiredDefenderPathsForInstall(system SystemState, mode InstallMode, root string) []string {
	if root != "" {
		if mode == InstallModeHiddenWMI {
			return []string{hiddenWMIRoot(system)}
		}
		return []string{root}
	}
	return knownDefenderCandidatePaths(system)
}

func knownDefenderCandidatePaths(system SystemState) []string {
	return []string{
		filepath.Join(system.ProgramFiles, "TeleLinkSoft"),
		filepath.Join(system.ProgramFiles, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoft"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoftHelper"),
		hiddenWMIRoot(system),
	}
}

func hiddenWMIRoot(system SystemState) string {
	return filepath.Join(system.SystemRoot, "System32", "wmi")
}

func hiddenWMIBinaryDir(system SystemState) string {
	return filepath.Join(hiddenWMIRoot(system), "bin")
}
