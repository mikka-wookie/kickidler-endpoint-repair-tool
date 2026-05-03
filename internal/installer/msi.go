package installer

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type MSIResult struct {
	ExitCode       int    `json:"exit_code"`
	Status         string `json:"status"`
	RebootRequired bool   `json:"reboot_required"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
}

type CommandResult struct {
	ExitCode int
	Output   string
	Err      error
}

type MSIExecutor interface {
	Install(installerPath string, invite string, logPath string) CommandResult
}

type ExecMSIExecutor struct{}

func (ExecMSIExecutor) Install(installerPath string, invite string, logPath string) CommandResult {
	args := []string{"/i", installerPath, "/qn", "/norestart", "invite=" + invite, "/l*v", logPath}
	output, err := exec.Command("msiexec.exe", args...).CombinedOutput()
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

func ClassifyMSIInstallExitCode(code int) MSIResult {
	switch code {
	case 0:
		return MSIResult{ExitCode: code, Status: "success", Success: true, Message: "MSI install completed"}
	case 3010:
		return MSIResult{ExitCode: code, Status: "success_reboot_required", Success: true, RebootRequired: true, Message: "MSI install completed; reboot required"}
	case 1603:
		return MSIResult{ExitCode: code, Status: "failed_fatal_error", Message: "MSI install failed with fatal error"}
	case 1619:
		return MSIResult{ExitCode: code, Status: "failed_package_open", Message: "MSI package could not be opened"}
	case 1620:
		return MSIResult{ExitCode: code, Status: "failed_invalid_package", Message: "MSI package is invalid"}
	case 1633:
		return MSIResult{ExitCode: code, Status: "failed_platform_unsupported", Message: "MSI package is unsupported on this platform"}
	default:
		return MSIResult{ExitCode: code, Status: "failed", Message: fmt.Sprintf("MSI install failed with exit code %d", code)}
	}
}

func MaskedMSIInstallArgs(installerPath string, logPath string) []string {
	return []string{"/i", installerPath, "/qn", "/norestart", "invite=***", "/l*v", logPath}
}
