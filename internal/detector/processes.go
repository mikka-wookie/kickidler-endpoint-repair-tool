package detector

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
)

type ProcessState struct {
	PID                int    `json:"pid"`
	Name               string `json:"name"`
	ExecutablePath     string `json:"executable_path,omitempty"`
	MatchedByName      bool   `json:"matched_by_name"`
	MatchedByExactPath bool   `json:"matched_by_exact_path"`
}

type cimProcess struct {
	ProcessID      int    `json:"ProcessId"`
	Name           string `json:"Name"`
	ExecutablePath string `json:"ExecutablePath"`
}

func DetectProcesses(system SystemState) []ProcessState {
	names := map[string]bool{}
	for _, name := range knownProcessNames() {
		names[strings.ToLower(name)] = true
	}
	wmiPaths := map[string]bool{}
	for _, path := range requiredWMIFilePaths(system) {
		wmiPaths[normalizePath(path)] = true
	}

	processes, err := queryProcesses()
	if err != nil {
		return nil
	}
	matched := make([]ProcessState, 0)
	for _, process := range processes {
		byName := names[strings.ToLower(process.Name)]
		byPath := wmiPaths[normalizePath(process.ExecutablePath)]
		if !byName && !byPath {
			continue
		}
		matched = append(matched, ProcessState{
			PID:                process.ProcessID,
			Name:               process.Name,
			ExecutablePath:     process.ExecutablePath,
			MatchedByName:      byName,
			MatchedByExactPath: byPath,
		})
	}
	return matched
}

func queryProcesses() ([]cimProcess, error) {
	script := `Get-CimInstance Win32_Process | Select-Object ProcessId,Name,ExecutablePath | ConvertTo-Json -Depth 2`
	output, err := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).Output()
	if err != nil {
		return nil, err
	}
	data := strings.TrimSpace(string(output))
	if data == "" {
		return nil, nil
	}
	var list []cimProcess
	if strings.HasPrefix(data, "{") {
		var one cimProcess
		if err := json.Unmarshal([]byte(data), &one); err != nil {
			return nil, err
		}
		return []cimProcess{one}, nil
	}
	if err := json.Unmarshal([]byte(data), &list); err != nil {
		return nil, err
	}
	return list, nil
}

func knownProcessNames() []string {
	return append([]string{}, standardExecutableCandidates...)
}

func requiredWMIFilePaths(system SystemState) []string {
	base := filepath.Join(system.SystemRoot, "System32", "wmi", "bin")
	return []string{
		filepath.Join(base, "svchost.exe"),
		filepath.Join(base, "WmiPrvSE.exe"),
		filepath.Join(base, "RuntimeBroker.exe"),
	}
}
