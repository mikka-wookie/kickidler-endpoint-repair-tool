package defender

import (
	"strconv"
	"strings"

	"kigrepair/internal/failures"
	"kigrepair/internal/winapi"
)

type CommandResult struct {
	ExitCode int
	Output   string
	Err      error
	TimedOut bool
}

type ExclusionAdder interface {
	AddExclusion(path string) CommandResult
}

type PowerShellExclusionAdder struct{}

func (PowerShellExclusionAdder) AddExclusion(path string) CommandResult {
	script := "Add-MpPreference -ExclusionPath " + quotePowerShellString(path)
	result := winapi.RunCommand(winapi.CommandOptions{
		Name:       "powershell.exe",
		Args:       []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script},
		Timeout:    winapi.DefaultPowerShellTimeout,
		RedactArgs: true,
		Category:   failures.FailureDefenderAccess,
	})
	return CommandResult{ExitCode: result.ExitCode, Output: result.CombinedOutput(), Err: commandErr(result), TimedOut: result.TimedOut}
}

func quotePowerShellString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func commandErrorSummary(result CommandResult) string {
	if result.TimedOut {
		return "Defender PowerShell command timed out"
	}
	if result.Output != "" {
		return strings.TrimSpace(result.Output)
	}
	if result.Err != nil {
		return result.Err.Error()
	}
	if result.ExitCode != 0 {
		return "exit code " + strconv.Itoa(result.ExitCode)
	}
	return ""
}

func commandErr(result winapi.CommandResult) error {
	if result.Error == "" {
		return nil
	}
	return &commandFailure{text: result.Error}
}

type commandFailure struct {
	text string
}

func (e *commandFailure) Error() string {
	return e.text
}
