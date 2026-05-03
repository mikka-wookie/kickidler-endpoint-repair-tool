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
	targets := make([]FileState, 0)
	seen := map[string]bool{}
	add := func(path, typ string) {
		if path == "." || path == "" || seen[normalizePath(path)] {
			return
		}
		seen[normalizePath(path)] = true
		targets = append(targets, FileState{Path: path, Type: typ})
	}
	for _, root := range knownInstallRoots(system) {
		mode := modeForInstallRoot(system, root)
		binaryDir := binaryDirForMode(mode, root)
		add(root, "folder")
		if !pathsEqual(binaryDir, root) {
			add(binaryDir, "folder")
		}
		for _, exe := range executableCandidatesForMode(mode) {
			add(filepath.Join(binaryDir, exe), "file")
		}
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

func ensureFileState(files []FileState, path string) []FileState {
	if path == "" {
		return files
	}
	for _, file := range files {
		if pathsEqual(file.Path, path) {
			return files
		}
	}
	return append(files, statFileState(FileState{Path: path, Type: "file"}))
}
