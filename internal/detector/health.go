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
	report = EnrichDetectionReport(report)
	var issues []string
	var recommendations []string

	if majorDetectionFailures(report) {
		return GrabberHealthUnknown,
			[]string{"Major detection failures prevent classification"},
			[]string{"Review repair.log and run check again from an elevated shell"}
	}

	primaryService := findService(report.Services, report.PrimaryService)
	primaryServiceExists := primaryService != nil && primaryService.Exists
	primaryServiceRunning := primaryServiceExists && strings.EqualFold(primaryService.Status, "running")
	requiredFiles := requiredFilePathsForInstall(report)
	missingRequiredFiles := missingFiles(report.Files, requiredFiles)
	allRequiredFilesExist := len(missingRequiredFiles) == 0
	anyServiceExists := anyExistingService(report.Services)
	anyFolderExists := anyExistingFileType(report.Files, "folder")
	anyExecutableExists := anyExistingFileType(report.Files, "file")
	anyRegistryExists := anyExistingRegistry(report.Registry)

	if !anyServiceExists && !anyFolderExists && !anyExecutableExists && !anyRegistryExists {
		return GrabberHealthNotInstalled, issues, recommendations
	}

	for _, path := range missingRequiredFiles {
		issues = append(issues, "Missing required file: "+path)
	}
	if primaryServiceExists && !primaryServiceRunning {
		issues = append(issues, "Service "+primaryService.Name+" exists but is not running")
	}
	if report.Defender.Available && report.InstallRoot != "" {
		for _, path := range report.MissingDefenderPaths {
			issues = append(issues, "Missing Defender exclusion: "+path)
		}
	}
	if !report.Defender.Available {
		issues = append(issues, "Defender exclusions could not be verified")
	}

	if anyServiceExists && report.PrimaryService == "" {
		issues = append(issues, "Known service exists but install root could not be resolved")
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --yes")
		return GrabberHealthBroken, issues, recommendations
	}

	if primaryServiceExists && report.ServiceExecutablePath == "" {
		issues = append(issues, "Service "+primaryService.Name+" ImagePath could not be parsed")
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --yes")
		return GrabberHealthBroken, issues, recommendations
	}

	if primaryServiceExists && report.InstallRoot == "" {
		issues = append(issues, "Service "+primaryService.Name+" install root could not be resolved")
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --yes")
		return GrabberHealthBroken, issues, recommendations
	}

	if primaryServiceExists && !allRequiredFilesExist {
		recommendations = appendRecommendation(recommendations, "Run: kigrepair.exe repair --invite <INVITE> --yes")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthBroken, issues, recommendations
	}

	if report.Defender.Available && report.InstallRoot != "" && len(report.MissingDefenderPaths) > 0 {
		recommendations = appendRecommendation(recommendations, "Run defender ensure or repair workflow to add the exclusion")
	}

	if primaryServiceExists && primaryServiceRunning && allRequiredFilesExist && report.InstallRoot != "" {
		return GrabberHealthHealthy, issues, recommendations
	}

	if !anyServiceExists && anyExecutableExists {
		issues = append(issues, "Known executable files remain without service")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthPartiallyRemoved, issues, recommendations
	}

	if !anyServiceExists && anyRegistryExists {
		issues = append(issues, "Registry leftovers detected without service")
		recommendations = appendRecommendation(recommendations, "Run cleanup if reinstall fails")
		return GrabberHealthPartiallyRemoved, issues, recommendations
	}

	if !anyServiceExists && anyFolderExists {
		issues = append(issues, "Known folders remain without service")
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
