package checks

import (
	"fmt"
	"os"
	"os/user"
	"runtime"

	"golang.org/x/sys/windows"
)

func OSName() string {
	return runtime.GOOS
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

func DetectSystemState() SystemState {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if current, err := user.Current(); err == nil && current.Username != "" {
		username = current.Username
	}
	return SystemState{
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		Windows:         WindowsVersion(),
		Hostname:        hostname,
		Username:        username,
		SystemRoot:      os.Getenv("SystemRoot"),
		ProgramFiles:    os.Getenv("ProgramFiles"),
		ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
		ProgramData:     os.Getenv("ProgramData"),
	}
}

func WindowsVersion() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	version := windows.RtlGetVersion()
	if version == nil {
		return os.Getenv("OS")
	}
	return fmt.Sprintf("%d.%d.%d", version.MajorVersion, version.MinorVersion, version.BuildNumber)
}
