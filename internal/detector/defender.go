package detector

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
)

type DefenderState struct {
	Available      bool     `json:"available"`
	ExclusionPaths []string `json:"exclusion_paths"`
	RequiredPaths  []string `json:"required_paths"`
	MissingPaths   []string `json:"missing_paths"`
	Error          string   `json:"error,omitempty"`
}

func DetectDefender(system SystemState) DefenderState {
	state := DefenderState{
		RequiredPaths: requiredDefenderPaths(system),
	}
	paths, err := queryDefenderExclusions()
	if err != nil {
		state.Available = false
		state.Error = err.Error()
		state.MissingPaths = append([]string{}, state.RequiredPaths...)
		return state
	}
	state.Available = true
	state.ExclusionPaths = paths
	state.MissingPaths = MissingPaths(state.RequiredPaths, paths)
	return state
}

func requiredDefenderPaths(system SystemState) []string {
	return []string{
		filepath.Join(system.ProgramFiles, "TeleLinkSoft"),
		filepath.Join(system.ProgramFiles, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoft"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoftHelper"),
		filepath.Join(system.SystemRoot, "System32", "wmi"),
	}
}

func queryDefenderExclusions() ([]string, error) {
	script := `(Get-MpPreference).ExclusionPath | ConvertTo-Json -Depth 2`
	output, err := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).CombinedOutput()
	if err != nil {
		if message := strings.TrimSpace(string(output)); message != "" {
			return nil, &commandError{message: message}
		}
		return nil, err
	}
	data := strings.TrimSpace(string(output))
	if data == "" || data == "null" {
		return nil, nil
	}
	var list []string
	if strings.HasPrefix(data, `"`) {
		var one string
		if err := json.Unmarshal([]byte(data), &one); err != nil {
			return nil, err
		}
		return []string{one}, nil
	}
	if err := json.Unmarshal([]byte(data), &list); err != nil {
		return nil, err
	}
	return list, nil
}

type commandError struct {
	message string
}

func (e *commandError) Error() string {
	return e.message
}

func MissingPaths(required, actual []string) []string {
	found := make(map[string]bool, len(actual))
	for _, path := range actual {
		found[normalizePath(path)] = true
	}
	var missing []string
	for _, path := range required {
		if !found[normalizePath(path)] {
			missing = append(missing, path)
		}
	}
	return missing
}
