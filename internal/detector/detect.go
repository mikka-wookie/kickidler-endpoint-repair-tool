package detector

import (
	"os"
	"os/user"
	"runtime"
	"time"

	"kigrepair/internal/winapi"
)

type DetectionReport struct {
	GeneratedAt             time.Time           `json:"generated_at"`
	IsAdmin                 bool                `json:"is_admin"`
	System                  SystemState         `json:"system"`
	InstallMode             InstallMode         `json:"install_mode"`
	InstallRoot             string              `json:"install_root,omitempty"`
	BinaryDir               string              `json:"binary_dir,omitempty"`
	ServiceSource           string              `json:"service_source,omitempty"`
	PrimaryService          string              `json:"primary_service,omitempty"`
	PrimaryServiceImagePath string              `json:"primary_service_image_path,omitempty"`
	ServiceExecutablePath   string              `json:"service_executable_path,omitempty"`
	ServiceExecutableExists bool                `json:"service_executable_exists"`
	RequiredDefenderPaths   []string            `json:"required_defender_paths,omitempty"`
	MissingDefenderPaths    []string            `json:"missing_defender_paths,omitempty"`
	Services                []ServiceState      `json:"services"`
	Files                   []FileState         `json:"files"`
	Processes               []ProcessState      `json:"processes"`
	Registry                []RegistryState     `json:"registry"`
	Defender                DefenderState       `json:"defender"`
	Health                  GrabberHealthStatus `json:"health"`
	Issues                  []string            `json:"issues"`
	Recommendations         []string            `json:"recommendations"`
}

type SystemState struct {
	GOOS            string `json:"goos"`
	GOARCH          string `json:"goarch"`
	Windows         string `json:"windows_version,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	Username        string `json:"username,omitempty"`
	SystemRoot      string `json:"system_root,omitempty"`
	ProgramFiles    string `json:"program_files,omitempty"`
	ProgramFilesX86 string `json:"program_files_x86,omitempty"`
	ProgramData     string `json:"program_data,omitempty"`
}

func Detect() DetectionReport {
	system := detectSystemState()
	report := DetectionReport{
		GeneratedAt: time.Now(),
		IsAdmin:     isAdmin(),
		System:      system,
	}
	report.Services = DetectServices(system)
	report.Files = DetectFiles(system)
	report.Processes = DetectProcesses(system)
	report.Registry = DetectRegistry()
	report = EnrichDetectionReport(report)
	report.Files = ensureFileState(report.Files, report.ServiceExecutablePath)
	report = EnrichDetectionReport(report)
	report.Defender = DetectDefender(system, report.InstallMode, report.InstallRoot)
	report = EnrichDetectionReport(report)
	report.Health, report.Issues, report.Recommendations = CalculateHealth(report)
	return report
}

func detectSystemState() SystemState {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if current, err := user.Current(); err == nil && current.Username != "" {
		username = current.Username
	}
	return SystemState{
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		Windows:         winapi.WindowsVersion(),
		Hostname:        hostname,
		Username:        username,
		SystemRoot:      os.Getenv("SystemRoot"),
		ProgramFiles:    os.Getenv("ProgramFiles"),
		ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
		ProgramData:     os.Getenv("ProgramData"),
	}
}

func isAdmin() bool {
	return winapi.IsAdmin()
}
