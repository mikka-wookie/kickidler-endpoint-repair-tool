package detector

import "strings"

type GrabberHealthStatus string

const (
	GrabberHealthHealthy          GrabberHealthStatus = "healthy"
	GrabberHealthBroken           GrabberHealthStatus = "broken"
	GrabberHealthPartiallyRemoved GrabberHealthStatus = "partially_removed"
	GrabberHealthNotInstalled     GrabberHealthStatus = "not_installed"
	GrabberHealthUnknown          GrabberHealthStatus = "unknown"
)

func CalculateHealth(report DetectionReport) (GrabberHealthStatus, []string, []string) {
	var issues []string
	var recommendations []string

	if majorDetectionFailures(report) {
		return GrabberHealthUnknown,
			[]string{"Major detection failures prevent classification"},
			[]string{"Review repair.log and run check again from an elevated shell"}
	}

	wmiService := findService(report.Services, wmiProviderService)
	wmiServiceExists := wmiService != nil && wmiService.Exists
	wmiServiceRunning := wmiServiceExists && strings.EqualFold(wmiService.Status, "running")
	wmiImagePath := expectedWMIServiceImagePath(report.System)
	requiredFiles := requiredWMIFilePaths(report.System)
	missingRequiredFiles := missingFiles(report.Files, requiredFiles)
	allRequiredFilesExist := len(missingRequiredFiles) == 0
	wmiExclusionMissing := containsPath(report.Defender.MissingPaths, filepathDir(wmiImagePath))
	anyServiceExists := anyExistingService(report.Services)
	anyFolderExists := anyExistingFileType(report.Files, "folder")
	anyRegistryExists := anyExistingRegistry(report.Registry)
	msiRegistryExists := anyExistingMSIRegistry(report.Registry)

	if !anyServiceExists && !anyFolderExists && !anyRegistryExists {
		return GrabberHealthNotInstalled, issues, recommendations
	}

	for _, path := range missingRequiredFiles {
		issues = append(issues, "Missing required file: "+path)
	}
	if wmiServiceExists && !wmiServiceRunning {
		issues = append(issues, "WmiProviderSE service exists but is not running")
	}
	if report.Defender.Available && wmiExclusionMissing {
		issues = append(issues, "Missing Defender exclusion: "+filepathDir(wmiImagePath))
	}
	if !report.Defender.Available {
		issues = append(issues, "Defender exclusion could not be verified")
		recommendations = appendRecommendation(recommendations, "Verify Defender exclusions manually")
	}

	if wmiServiceExists && wmiServiceRunning && wmiService.ExpectedImagePathMatch && allRequiredFilesExist {
		if !report.Defender.Available || !wmiExclusionMissing {
			return GrabberHealthHealthy, issues, recommendations
		}
	}

	if wmiServiceExists && len(missingRequiredFiles) > 0 {
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --installer .\\grabber.msi")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthBroken, issues, recommendations
	}
	if wmiServiceExists && wmiService.ExpectedImagePathMatch && !fileExists(report.Files, wmiImagePath) {
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --installer .\\grabber.msi")
		return GrabberHealthBroken, issues, recommendations
	}
	if msiRegistryExists && len(missingRequiredFiles) > 0 {
		issues = append(issues, "MSI registry entries exist but required files are missing")
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --installer .\\grabber.msi")
		return GrabberHealthBroken, issues, recommendations
	}
	if report.Defender.Available && wmiExclusionMissing && len(missingRequiredFiles) > 0 {
		recommendations = appendRecommendation(recommendations, "Verify Defender exclusions manually")
		return GrabberHealthBroken, issues, recommendations
	}

	if !anyServiceExists && anyFolderExists {
		issues = append(issues, "Known folders remain without service")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthPartiallyRemoved, issues, recommendations
	}
	if !anyServiceExists && anyRegistryExists {
		issues = append(issues, "Registry leftovers detected without service")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthPartiallyRemoved, issues, recommendations
	}
	if !wmiServiceExists && (fileExists(report.Files, filepathDir(wmiImagePath)) || programDataFolderExists(report)) {
		issues = append(issues, "WMI or ProgramData remnants detected without service")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthPartiallyRemoved, issues, recommendations
	}

	return GrabberHealthUnknown, issues, recommendations
}

func majorDetectionFailures(report DetectionReport) bool {
	for _, service := range report.Services {
		if service.Error != "" {
			return true
		}
	}
	registryErrors := 0
	for _, key := range report.Registry {
		if key.Error != "" {
			registryErrors++
		}
	}
	return len(report.Registry) > 0 && registryErrors == len(report.Registry)
}

func findService(services []ServiceState, name string) *ServiceState {
	for i := range services {
		if strings.EqualFold(services[i].Name, name) {
			return &services[i]
		}
	}
	return nil
}

func anyExistingService(services []ServiceState) bool {
	for _, service := range services {
		if service.Exists {
			return true
		}
	}
	return false
}

func anyExistingFileType(files []FileState, typ string) bool {
	for _, file := range files {
		if file.Exists && file.Type == typ {
			return true
		}
	}
	return false
}

func anyExistingRegistry(keys []RegistryState) bool {
	for _, key := range keys {
		if key.Exists {
			return true
		}
	}
	return false
}

func anyExistingMSIRegistry(keys []RegistryState) bool {
	for _, key := range keys {
		if key.Exists && (strings.Contains(key.Path, `Installer\`) || strings.Contains(key.Path, `Uninstall\{EB1FBC37`)) {
			return true
		}
	}
	return false
}

func missingFiles(files []FileState, paths []string) []string {
	var missing []string
	for _, path := range paths {
		if !fileExists(files, path) {
			missing = append(missing, path)
		}
	}
	return missing
}

func fileExists(files []FileState, path string) bool {
	for _, file := range files {
		if pathsEqual(file.Path, path) {
			return file.Exists
		}
	}
	return false
}

func containsPath(paths []string, path string) bool {
	for _, candidate := range paths {
		if pathsEqual(candidate, path) {
			return true
		}
	}
	return false
}

func filepathDir(path string) string {
	index := strings.LastIndex(strings.ReplaceAll(path, "/", `\`), `\`)
	if index < 0 {
		return path
	}
	return path[:index]
}

func programDataFolderExists(report DetectionReport) bool {
	target := report.System.ProgramData + `\E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60`
	return fileExists(report.Files, target)
}

func appendRecommendation(recommendations []string, recommendation string) []string {
	for _, existing := range recommendations {
		if existing == recommendation {
			return recommendations
		}
	}
	return append(recommendations, recommendation)
}
