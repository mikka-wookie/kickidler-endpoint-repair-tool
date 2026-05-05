package detector

import (
	"encoding/json"
	"strings"

	"kigrepair/internal/failures"
	"kigrepair/internal/winapi"
)

type DefenderState struct {
	Available                bool     `json:"available"`
	ExclusionPaths           []string `json:"exclusion_paths"`
	NormalizedExclusionPaths []string `json:"normalized_exclusion_paths,omitempty"`
	RequiredPaths            []string `json:"required_paths"`
	CandidatePaths           []string `json:"candidate_required_paths,omitempty"`
	MissingPaths             []string `json:"missing_paths"`
	CoveredPaths             []string `json:"covered_paths"`
	Error                    string   `json:"error,omitempty"`
}

func DetectDefender(system SystemState, mode InstallMode, installRoot string) DefenderState {
	state := DefenderState{
		ExclusionPaths:           []string{},
		NormalizedExclusionPaths: []string{},
		RequiredPaths:            requiredDefenderPathsForInstall(system, mode, installRoot),
		MissingPaths:             []string{},
		CoveredPaths:             []string{},
	}
	if installRoot == "" {
		state.CandidatePaths = append([]string{}, state.RequiredPaths...)
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
	state.NormalizedExclusionPaths, state.CoveredPaths, state.MissingPaths = EvaluateDefenderCoverage(state.RequiredPaths, paths)
	return state
}

func queryDefenderExclusions() ([]string, error) {
	script := `$p=(Get-MpPreference).ExclusionPath; if ($null -eq $p) { @() | ConvertTo-Json -Compress } else { @($p) | ConvertTo-Json -Compress }`
	result := winapi.RunCommand(winapi.CommandOptions{
		Name:       "powershell.exe",
		Args:       []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script},
		Timeout:    winapi.DefaultPowerShellTimeout,
		RedactArgs: true,
		Category:   failures.FailureDefenderAccess,
	})
	if result.ExitCode != 0 {
		if message := strings.TrimSpace(result.CombinedOutput()); message != "" {
			return nil, &commandError{message: message}
		}
		return nil, &commandError{message: result.Error}
	}
	data := strings.TrimSpace(result.CombinedOutput())
	return parseDefenderExclusionJSON(data)
}

func parseDefenderExclusionJSON(data string) ([]string, error) {
	if data == "" || data == "null" || data == "[]" {
		return nil, nil
	}
	var list []string
	if strings.HasPrefix(data, `"`) {
		var one string
		if err := json.Unmarshal([]byte(data), &one); err != nil {
			return nil, err
		}
		return validateDefenderExclusionPaths([]string{one})
	}
	if err := json.Unmarshal([]byte(data), &list); err != nil {
		return nil, err
	}
	return validateDefenderExclusionPaths(list)
}

func validateDefenderExclusionPaths(paths []string) ([]string, error) {
	for _, path := range paths {
		if isDefenderUnavailableValue(path) {
			return nil, &commandError{message: path}
		}
	}
	return paths, nil
}

func isDefenderUnavailableValue(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "n/a:") || strings.Contains(lower, "administrator to view exclusions")
}

type commandError struct {
	message string
}

func (e *commandError) Error() string {
	return e.message
}

func MissingPaths(required, actual []string) []string {
	_, _, missing := EvaluateDefenderCoverage(required, actual)
	return missing
}

func EvaluateDefenderCoverage(required, exclusions []string) ([]string, []string, []string) {
	normalized := make([]string, 0, len(exclusions))
	for _, path := range exclusions {
		normalized = append(normalized, NormalizeWindowsPath(path))
	}
	covered := make([]string, 0, len(required))
	missing := make([]string, 0, len(required))
	for _, path := range required {
		if IsPathCoveredByAnyExclusion(path, exclusions) {
			covered = append(covered, path)
		} else {
			missing = append(missing, path)
		}
	}
	return normalized, covered, missing
}

func IsPathCoveredByAnyExclusion(requiredPath string, exclusionPaths []string) bool {
	for _, exclusionPath := range exclusionPaths {
		if IsPathCoveredByExclusion(requiredPath, exclusionPath) {
			return true
		}
	}
	return false
}

func IsPathCoveredByExclusion(requiredPath string, exclusionPath string) bool {
	required := strings.ToLower(NormalizeWindowsPath(requiredPath))
	exclusion := strings.ToLower(NormalizeWindowsPath(exclusionPath))
	if required == "" || exclusion == "" || required == "." || exclusion == "." {
		return false
	}
	if required == exclusion {
		return true
	}
	if strings.HasSuffix(exclusion, `\`) {
		return strings.HasPrefix(required, exclusion)
	}
	return strings.HasPrefix(required, exclusion+`\`)
}
