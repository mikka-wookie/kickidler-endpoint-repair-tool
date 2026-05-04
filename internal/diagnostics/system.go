package diagnostics

import (
	"os"
	"runtime"
	"time"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type SystemInfo struct {
	Timestamp        time.Time `json:"timestamp"`
	Hostname         string    `json:"hostname,omitempty"`
	Username         string    `json:"username,omitempty"`
	IsAdmin          bool      `json:"is_admin"`
	GOOS             string    `json:"goos"`
	GOARCH           string    `json:"goarch"`
	WindowsVersion   string    `json:"windows_version,omitempty"`
	SystemRoot       string    `json:"system_root,omitempty"`
	ProgramFiles     string    `json:"program_files,omitempty"`
	ProgramFilesX86  string    `json:"program_files_x86,omitempty"`
	ProgramData      string    `json:"program_data,omitempty"`
	Temp             string    `json:"temp,omitempty"`
	WorkingDirectory string    `json:"working_directory,omitempty"`
	ExecutablePath   string    `json:"executable_path,omitempty"`
}

type SystemInfoCollector struct {
	Report detector.DetectionReport
}

func (SystemInfoCollector) Name() string {
	return "system"
}

func (c SystemInfoCollector) Collect(ctx *app.AppContext) app.OperationResult {
	wd, _ := os.Getwd()
	exe, _ := os.Executable()
	info := SystemInfo{
		Timestamp:        time.Now(),
		Hostname:         c.Report.System.Hostname,
		Username:         c.Report.System.Username,
		IsAdmin:          c.Report.IsAdmin,
		GOOS:             runtime.GOOS,
		GOARCH:           runtime.GOARCH,
		WindowsVersion:   c.Report.System.Windows,
		SystemRoot:       os.Getenv("SystemRoot"),
		ProgramFiles:     os.Getenv("ProgramFiles"),
		ProgramFilesX86:  os.Getenv("ProgramFiles(x86)"),
		ProgramData:      os.Getenv("ProgramData"),
		Temp:             os.Getenv("TEMP"),
		WorkingDirectory: wd,
		ExecutablePath:   exe,
	}
	return writeJSON(ctx, "system", info)
}
