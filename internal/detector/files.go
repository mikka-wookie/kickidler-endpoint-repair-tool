package detector

import (
	"os"
	"path/filepath"
	"time"
)

type FileState struct {
	Path         string    `json:"path"`
	Exists       bool      `json:"exists"`
	Type         string    `json:"type"`
	Size         int64     `json:"size,omitempty"`
	ModifiedTime time.Time `json:"modified_time,omitempty"`
	Error        string    `json:"error,omitempty"`
}

func DetectFiles(system SystemState) []FileState {
	targets := []FileState{
		{Path: filepath.Join(system.ProgramFiles, "TeleLinkSoft"), Type: "folder"},
		{Path: filepath.Join(system.ProgramFiles, "TeleLinkSoftHelper"), Type: "folder"},
		{Path: filepath.Join(system.ProgramFilesX86, "TeleLinkSoft"), Type: "folder"},
		{Path: filepath.Join(system.ProgramFilesX86, "TeleLinkSoftHelper"), Type: "folder"},
		{Path: filepath.Join(system.ProgramData, "E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60"), Type: "folder"},
		{Path: filepath.Join(system.SystemRoot, "System32", "wmi"), Type: "folder"},
		{Path: filepath.Join(system.SystemRoot, "System32", "wmi", "bin"), Type: "folder"},
		{Path: filepath.Join(system.SystemRoot, "System32", "wmi", "bin", "svchost.exe"), Type: "file"},
		{Path: filepath.Join(system.SystemRoot, "System32", "wmi", "bin", "WmiPrvSE.exe"), Type: "file"},
		{Path: filepath.Join(system.SystemRoot, "System32", "wmi", "bin", "RuntimeBroker.exe"), Type: "file"},
	}
	for i := range targets {
		targets[i] = statFileState(targets[i])
	}
	return targets
}

func statFileState(state FileState) FileState {
	info, err := os.Stat(state.Path)
	if err != nil {
		if os.IsNotExist(err) {
			state.Exists = false
			return state
		}
		state.Error = err.Error()
		return state
	}
	state.Exists = true
	state.ModifiedTime = info.ModTime()
	if info.IsDir() {
		state.Type = "folder"
	} else {
		state.Type = "file"
		state.Size = info.Size()
	}
	return state
}
