package detector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"kigrepair/internal/config"
	"kigrepair/internal/failures"
	"kigrepair/internal/winapi"
)

type ProcessState struct {
	PID                      int      `json:"pid"`
	Name                     string   `json:"name"`
	ExecutablePath           string   `json:"executable_path,omitempty"`
	NormalizedExecutablePath string   `json:"normalized_executable_path,omitempty"`
	CommandLine              string   `json:"command_line,omitempty"`
	InstallRoot              string   `json:"install_root,omitempty"`
	InstallMode              string   `json:"install_mode,omitempty"`
	GrabberRelated           bool     `json:"grabber_related"`
	MatchReason              string   `json:"match_reason,omitempty"`
	TrustLevel               string   `json:"trust_level"`
	CanTerminate             bool     `json:"can_terminate"`
	Warnings                 []string `json:"warnings,omitempty"`
	Errors                   []string `json:"errors,omitempty"`
	MatchedByName            bool     `json:"matched_by_name"`
	MatchedByExactPath       bool     `json:"matched_by_exact_path"`
}

type RawProcessInfo struct {
	ProcessID      int    `json:"ProcessId"`
	Name           string `json:"Name"`
	ExecutablePath string `json:"ExecutablePath"`
	CommandLine    string `json:"CommandLine"`
}

type MatchOptions struct {
	System SystemState
}

const (
	ProcessTrustTrusted            = "trusted"
	ProcessTrustNameAndPathMatch   = "name_and_path_match"
	ProcessTrustHiddenWMIExactPath = "hidden_wmi_exact_path"
	ProcessTrustNameOnlyUntrusted  = "name_only_untrusted"
	ProcessTrustPathMismatch       = "path_mismatch"
	ProcessTrustPathUnavailable    = "path_unavailable"
	ProcessTrustUnknown            = "unknown"
	ProcessTrustQueryFailed        = "query_failed"
)

func DetectProcesses(system SystemState) []ProcessState {
	processes, err := queryProcesses()
	if err != nil {
		return []ProcessState{{
			TrustLevel: ProcessTrustQueryFailed,
			Errors:     []string{err.Error()},
		}}
	}
	matched := make([]ProcessState, 0)
	for _, process := range processes {
		classified := ClassifyProcess(process, MatchOptions{System: system})
		if !classified.GrabberRelated && len(classified.Warnings) == 0 {
			continue
		}
		matched = append(matched, classified)
	}
	return matched
}

func ClassifyProcess(p RawProcessInfo, opts MatchOptions) ProcessState {
	system := withDefaultSystem(opts.System)
	state := ProcessState{
		PID:                      p.ProcessID,
		Name:                     strings.TrimSpace(p.Name),
		ExecutablePath:           strings.TrimSpace(p.ExecutablePath),
		CommandLine:              strings.TrimSpace(p.CommandLine),
		NormalizedExecutablePath: NormalizeWindowsPath(p.ExecutablePath),
		TrustLevel:               ProcessTrustUnknown,
	}
	name := strings.ToLower(state.Name)
	path := state.NormalizedExecutablePath
	normalName := isKnownNormalProcessName(name)
	hiddenName := isKnownHiddenWMIProcessName(name)

	switch {
	case normalName:
		state.MatchedByName = true
		state.GrabberRelated = true
		if path == "" || path == "." {
			state.TrustLevel = ProcessTrustPathUnavailable
			state.MatchReason = "known Grabber process name but executable path is unavailable"
			state.Warnings = append(state.Warnings, "process path unavailable, not safe for automatic termination")
			return state
		}
		root, mode, ok := trustedNormalProcessRoot(system, path)
		if !ok {
			state.GrabberRelated = true
			state.TrustLevel = ProcessTrustPathMismatch
			state.MatchReason = "known Grabber process name outside allowlisted install roots"
			state.Warnings = append(state.Warnings, "known process name outside allowlisted Grabber path")
			return state
		}
		state.InstallRoot = root
		state.InstallMode = string(mode)
		state.TrustLevel = ProcessTrustNameAndPathMatch
		state.MatchReason = "known Grabber process name under allowlisted install root"
		state.CanTerminate = true
		return state
	case hiddenName:
		if path == "" || path == "." {
			state.GrabberRelated = false
			state.TrustLevel = ProcessTrustPathUnavailable
			state.MatchReason = "hidden WMI process name but executable path is unavailable"
			state.Warnings = append(state.Warnings, "hidden WMI process name without exact path match was skipped")
			return state
		}
		if isAllowedHiddenWMIProcessPath(system, state.Name, path) {
			state.GrabberRelated = true
			state.MatchedByExactPath = true
			state.InstallRoot = hiddenWMIRoot(system)
			state.InstallMode = string(InstallModeHiddenWMI)
			state.TrustLevel = ProcessTrustHiddenWMIExactPath
			state.MatchReason = "hidden WMI executable name exactly matches Grabber WMI allowlist path"
			state.CanTerminate = true
			return state
		}
		state.GrabberRelated = false
		state.TrustLevel = ProcessTrustPathMismatch
		state.MatchReason = "normal Windows process path, not Grabber hidden WMI path"
		state.Warnings = append(state.Warnings, "hidden WMI process name did not match exact Grabber WMI path")
		return state
	default:
		if pathInsideOrEqual(path, hiddenWMIBinaryDir(system)) || pathInsideOrEqual(path, hiddenWMIRoot(system)) {
			state.TrustLevel = ProcessTrustPathMismatch
			state.MatchReason = "process is under hidden WMI root but filename is not allowlisted"
			state.Warnings = append(state.Warnings, "process under hidden WMI root has unsupported executable name")
		}
		return state
	}
}

func queryProcesses() ([]RawProcessInfo, error) {
	if native, err := winapi.ListProcesses(context.Background()); err == nil {
		result := make([]RawProcessInfo, 0, len(native))
		for _, process := range native {
			result = append(result, RawProcessInfo{
				ProcessID:      process.ProcessID,
				Name:           process.Name,
				ExecutablePath: process.ExecutablePath,
				CommandLine:    process.CommandLine,
			})
		}
		return result, nil
	}
	script := `$procs = Get-CimInstance Win32_Process | Select-Object ProcessId,Name,ExecutablePath,CommandLine; @($procs) | ConvertTo-Json -Compress -Depth 3`
	result := winapi.RunCommand(winapi.CommandOptions{
		Name:       "powershell.exe",
		Args:       []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script},
		Timeout:    winapi.DefaultProcessQueryTimeout,
		RedactArgs: true,
		Category:   failures.FailureProcessControl,
	})
	if result.ExitCode != 0 {
		if message := strings.TrimSpace(result.CombinedOutput()); message != "" {
			return nil, errors.New(message)
		}
		return nil, errors.New(result.Error)
	}
	return ParseProcessJSON(result.CombinedOutput())
}

func ParseProcessJSON(data string) ([]RawProcessInfo, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, nil
	}
	if data == "null" {
		return nil, nil
	}
	if strings.HasPrefix(data, "{") {
		var one RawProcessInfo
		if err := json.Unmarshal([]byte(data), &one); err != nil {
			return nil, err
		}
		return []RawProcessInfo{one}, nil
	}
	var list []RawProcessInfo
	if err := json.Unmarshal([]byte(data), &list); err != nil {
		return nil, err
	}
	return list, nil
}

func knownProcessNames() []string {
	return append([]string{}, config.KnownProcessNames...)
}

func requiredWMIFilePaths(system SystemState) []string {
	base := filepath.Join(system.SystemRoot, "System32", "wmi", "bin")
	return []string{
		filepath.Join(base, "svchost.exe"),
		filepath.Join(base, "WmiPrvSE.exe"),
		filepath.Join(base, "RuntimeBroker.exe"),
	}
}

func isKnownNormalProcessName(name string) bool {
	for _, known := range config.KnownProcessNames {
		if strings.EqualFold(name, known) {
			return true
		}
	}
	return false
}

func isKnownHiddenWMIProcessName(name string) bool {
	for _, known := range hiddenWMIExecutableCandidates {
		if strings.EqualFold(name, known) {
			return true
		}
	}
	return false
}

func trustedNormalProcessRoot(system SystemState, exePath string) (string, InstallMode, bool) {
	for _, root := range normalProcessAllowedRoots(system) {
		if pathInsideOrEqual(exePath, root) {
			return processInstallRoot(system, root), modeForInstallRoot(system, root), true
		}
	}
	return "", InstallModeUnknown, false
}

func normalProcessAllowedRoots(system SystemState) []string {
	return []string{
		filepath.Join(system.ProgramFiles, "TeleLinkSoft"),
		filepath.Join(system.ProgramFiles, "TeleLinkSoft", "bin"),
		filepath.Join(system.ProgramFiles, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoft"),
		filepath.Join(system.ProgramFilesX86, "TeleLinkSoftHelper"),
		filepath.Join(system.ProgramData, "E891C8F2-6D3B-5E17-7F3C-9A1D4E2B8C60"),
	}
}

func processInstallRoot(system SystemState, root string) string {
	if pathInsideOrEqual(root, filepath.Join(system.ProgramFiles, "TeleLinkSoft", "bin")) {
		return filepath.Join(system.ProgramFiles, "TeleLinkSoft")
	}
	return NormalizeWindowsPath(root)
}

func isAllowedHiddenWMIProcessPath(system SystemState, name string, exePath string) bool {
	want := filepath.Join(hiddenWMIBinaryDir(system), name)
	return pathsEqual(exePath, want)
}

func pathInsideOrEqual(path string, root string) bool {
	path = normalizePath(path)
	root = normalizePath(root)
	if path == "" || root == "" || path == "." || root == "." {
		return false
	}
	return path == root || strings.HasPrefix(path, root+`\`)
}

func TrustedProcessForTermination(process ProcessState) bool {
	if !process.CanTerminate {
		return false
	}
	switch process.TrustLevel {
	case ProcessTrustNameAndPathMatch, ProcessTrustHiddenWMIExactPath, ProcessTrustTrusted:
		return true
	default:
		return false
	}
}

func withDefaultSystem(system SystemState) SystemState {
	if system.SystemRoot == "" {
		system.SystemRoot = config.ExpandPath(`%SystemRoot%`)
	}
	if system.ProgramFiles == "" {
		system.ProgramFiles = config.ExpandPath(`%ProgramFiles%`)
	}
	if system.ProgramFilesX86 == "" {
		system.ProgramFilesX86 = config.ExpandPath(`%ProgramFiles(x86)%`)
	}
	if system.ProgramData == "" {
		system.ProgramData = config.ExpandPath(`%ProgramData%`)
	}
	return system
}

func ProcessTarget(process ProcessState) string {
	target := fmt.Sprintf("%s PID %d", process.Name, process.PID)
	if strings.TrimSpace(process.ExecutablePath) != "" {
		target += " " + process.ExecutablePath
	}
	return target
}
