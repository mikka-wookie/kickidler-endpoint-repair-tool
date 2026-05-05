package installer

import (
	"fmt"
	"strings"

	"kigrepair/internal/failures"
	"kigrepair/internal/winapi"
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
	TimedOut bool
}

type MSIExecutor interface {
	Install(installerPath string, invite string, logPath string) CommandResult
}

type ExecMSIExecutor struct{}

func (ExecMSIExecutor) Install(installerPath string, invite string, logPath string) CommandResult {
	args := []string{"/i", installerPath, "/qn", "/norestart", "invite=" + invite, "/l*v", logPath}
	result := winapi.RunCommand(winapi.CommandOptions{
		Name:          "msiexec.exe",
		Args:          args,
		Timeout:       winapi.DefaultMSITimeout,
		SensitiveArgs: []string{invite},
		Category:      failures.FailureMSI,
	})
	return CommandResult{ExitCode: result.ExitCode, Output: result.CombinedOutput(), Err: commandErr(result), TimedOut: result.TimedOut}
}

func ClassifyMSIInstallExitCode(code int) MSIResult {
	switch code {
	case -1:
		return MSIResult{ExitCode: code, Status: "failed_timeout", Message: "MSI install timed out"}
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
	return []string{"/i", installerPath, "/qn", "/norestart", "invite=<REDACTED>", "/l*v", logPath}
}

func MaskInviteInText(text string, invite string) string {
	invite = strings.TrimSpace(invite)
	if invite == "" || text == "" {
		return text
	}
	masked := strings.ReplaceAll(text, invite, "<REDACTED>")
	masked = strings.ReplaceAll(masked, "invite=<REDACTED>", "invite=<REDACTED>")
	return masked
}

func commandErr(result winapi.CommandResult) error {
	if result.Error == "" {
		return nil
	}
	return fmt.Errorf("%s", result.Error)
}
