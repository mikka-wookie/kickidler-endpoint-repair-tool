package defender

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

type CommandResult struct {
	ExitCode int
	Output   string
	Err      error
}

type ExclusionAdder interface {
	AddExclusion(path string) CommandResult
}

type PowerShellExclusionAdder struct{}

func (PowerShellExclusionAdder) AddExclusion(path string) CommandResult {
	script := "Add-MpPreference -ExclusionPath " + quotePowerShellString(path)
	output, err := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).CombinedOutput()
	code := 0
	if err != nil {
		code = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
	}
	return CommandResult{ExitCode: code, Output: strings.TrimSpace(string(output)), Err: err}
}

func quotePowerShellString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func commandErrorSummary(result CommandResult) string {
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
