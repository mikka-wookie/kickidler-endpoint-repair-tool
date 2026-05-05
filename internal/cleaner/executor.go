package cleaner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"

	"kigrepair/internal/app"
	"kigrepair/internal/config"
	"kigrepair/internal/detector"
	"kigrepair/internal/logging"
	"kigrepair/internal/safety"
	svc "kigrepair/internal/services"
)

type CommandResult struct {
	ExitCode int
	Output   string
	Err      error
}

type CommandRunner func(name string, args ...string) CommandResult

type Executor struct {
	OutputDir       string
	Logger          app.Logger
	RunCommand      CommandRunner
	RemoveAll       func(string) error
	Stat            func(string) (os.FileInfo, error)
	ValidatePath    func(string) error
	RegistryBackups bool
	ProcessTimeout  time.Duration
	ProcessPoll     time.Duration
}

func NewExecutor(outputDir string, logger app.Logger) Executor {
	return Executor{
		OutputDir:       outputDir,
		Logger:          logger,
		RunCommand:      runCommand,
		RemoveAll:       os.RemoveAll,
		Stat:            os.Lstat,
		ValidatePath:    safety.ValidateCleanupPath,
		RegistryBackups: true,
		ProcessTimeout:  10 * time.Second,
		ProcessPoll:     500 * time.Millisecond,
	}
}

func (e Executor) ExecutePlan(plan CleanupPlan) []app.OperationResult {
	actions := SortActionsForExecution(plan.Actions)
	results := make([]app.OperationResult, 0, len(actions))
	for _, action := range actions {
		result := e.ExecuteAction(action)
		results = append(results, result)
		logging.LogOperation(e.Logger, result)
	}
	return results
}

func (e Executor) ExecuteAction(action CleanupAction) app.OperationResult {
	if !action.Safe {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Action is not marked safe", "unsafe cleanup action")
	}

	if e.Logger != nil {
		e.Logger.Info("starting cleanup action: %s target=%s", action.Type, action.Target)
	}

	switch action.Type {
	case CleanupActionStopService:
		return e.stopService(action)
	case CleanupActionKillProcess:
		return e.killProcess(action)
	case CleanupActionMSIUninstall:
		return e.uninstallMSI(action)
	case CleanupActionDeleteService:
		return e.deleteService(action)
	case CleanupActionDeletePath:
		return e.deletePath(action)
	case CleanupActionDeleteRegistryKey:
		return e.deleteRegistryKey(action)
	default:
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Unknown cleanup action", string(action.Type))
	}
}

func (e Executor) stopService(action CleanupAction) app.OperationResult {
	if e.Logger != nil {
		e.Logger.Info("executing command: sc.exe stop %s", action.Target)
	}
	started := time.Now()
	result := svc.StopService(context.Background(), action.Target, svc.StopOptions{Runner: serviceRunner(e.RunCommand)})
	e.logServiceAction(result, time.Since(started))
	return serviceOperation(action.Type, action.Target, result)
}

func (e Executor) deleteService(action CleanupAction) app.OperationResult {
	if e.Logger != nil {
		e.Logger.Info("executing command: sc.exe delete %s", action.Target)
	}
	started := time.Now()
	result := svc.DeleteService(context.Background(), action.Target, svc.DeleteOptions{Runner: serviceRunner(e.RunCommand), AllowStop: false})
	e.logServiceAction(result, time.Since(started))
	return serviceOperation(action.Type, action.Target, result)
}

func (e Executor) killProcess(action CleanupAction) app.OperationResult {
	// MVP implementation: uses taskkill.exe by PID. Keep this isolated so it can
	// later be replaced with native Windows process APIs.
	if !plannedProcessActionTrusted(action) {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Cleanup plan process action is not trusted for termination", action.TrustLevel)
	}
	pid := action.PID
	if pid == 0 {
		pid = parsePID(action.Target)
	}
	if pid <= 0 {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Cleanup plan does not contain a valid PID", "")
	}
	current, err := currentProcess(e.RunCommand, pid)
	if err != nil {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Could not re-query process before termination", err.Error())
	}
	if current == nil {
		return operation(action.Type, action.Target, app.OperationStatusSkipped, "Process already exited", "")
	}
	if err := validateCurrentProcessMatchesPlan(*current, action); err != nil {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Process no longer matches cleanup plan", err.Error())
	}
	if e.Logger != nil {
		e.Logger.Info("executing command: taskkill /PID %d /F", pid)
	}
	started := time.Now()
	result := e.RunCommand("taskkill.exe", "/PID", strconv.Itoa(pid), "/F")
	e.logCommandResult("taskkill.exe", result, time.Since(started))
	if result.ExitCode != 0 {
		message := "Process termination failed"
		if strings.Contains(strings.ToLower(result.Output), "access is denied") || strings.Contains(strings.ToLower(result.Output), "access denied") {
			message = "Process termination failed: run as Administrator"
		}
		return operation(action.Type, action.Target, app.OperationStatusFailed, message, resultError(result))
	}
	if waitErr := e.waitForProcessExit(pid); waitErr != nil {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Timed out waiting for process to exit", waitErr.Error())
	}
	return operation(action.Type, action.Target, app.OperationStatusSuccess, "Process terminated and verified exited", "")
}

func (e Executor) uninstallMSI(action CleanupAction) app.OperationResult {
	logPath := filepath.Join(e.OutputDir, "msi-uninstall.log")
	args := []string{"/x", config.MSIProductCode, "/qn", "/norestart", "/l*v", logPath}
	if e.Logger != nil {
		e.Logger.Info("executing command: msiexec %s", strings.Join(args, " "))
	}
	started := time.Now()
	result := e.RunCommand("msiexec.exe", args...)
	e.logCommandResult("msiexec.exe", result, time.Since(started))
	classification := ClassifyMSIUninstallExitCode(result.ExitCode)
	errText := ""
	if classification.Status == app.OperationStatusFailed {
		errText = resultError(result)
	}
	return operation(action.Type, action.Target, classification.Status, classification.Message, errText)
}

func (e Executor) deletePath(action CleanupAction) app.OperationResult {
	target := detector.NormalizeWindowsPath(action.Target)
	if e.Logger != nil {
		e.Logger.Info("validating cleanup path before deletion: %s", target)
	}
	if err := e.ValidatePath(target); err != nil {
		return operation(action.Type, target, app.OperationStatusFailed, "Cleanup path safety validation failed", err.Error())
	}
	info, err := e.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return operation(action.Type, target, app.OperationStatusSkipped, "Path does not exist", "")
		}
		return operation(action.Type, target, app.OperationStatusFailed, "Could not inspect path before deletion", err.Error())
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return operation(action.Type, target, app.OperationStatusFailed, "Refused to delete symlink cleanup path", "")
	}
	if err := e.RemoveAll(target); err != nil {
		return operation(action.Type, target, app.OperationStatusFailed, "Path deletion failed", err.Error())
	}
	if _, err := e.Stat(target); err == nil {
		return operation(action.Type, target, app.OperationStatusWarning, "Path deletion completed but target still exists", "")
	}
	return operation(action.Type, target, app.OperationStatusSuccess, "Path deleted", "")
}

func (e Executor) deleteRegistryKey(action CleanupAction) app.OperationResult {
	if !isAllowedRegistryTarget(action.Target) {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Registry key is not in the cleanup allowlist", "")
	}
	root, path, err := splitRegistryTarget(action.Target)
	if err != nil {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Registry target could not be parsed", err.Error())
	}
	if !registryKeyExists(root, path) {
		return operation(action.Type, action.Target, app.OperationStatusSkipped, "Registry key does not exist", "")
	}
	if e.RegistryBackups {
		if err := exportRegistryKey(e.RunCommand, e.OutputDir, action.Target); err != nil && e.Logger != nil {
			e.Logger.Warn("registry backup failed for %s: %v", action.Target, err)
		}
	} else if e.Logger != nil {
		e.Logger.Warn("Registry backup not implemented for this key: %s", action.Target)
	}
	if err := deleteRegistryTree(root, path); err != nil {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Registry key deletion failed", err.Error())
	}
	return operation(action.Type, action.Target, app.OperationStatusSuccess, "Registry key deleted", "")
}

func (e Executor) logCommandResult(name string, result CommandResult, duration time.Duration) {
	if e.Logger == nil {
		return
	}
	e.Logger.Info("%s exit code: %d", name, result.ExitCode)
	e.Logger.Info("%s duration: %s", name, duration.Round(time.Millisecond))
	if strings.TrimSpace(result.Output) != "" {
		e.Logger.Info("%s output summary: %s", name, shortCommandOutput(result.Output))
	}
	if result.Err != nil {
		e.Logger.Warn("%s error: %v", name, result.Err)
	}
}

func (e Executor) logServiceAction(result svc.ServiceActionResult, duration time.Duration) {
	if e.Logger == nil {
		return
	}
	e.Logger.Info("service %s %s status: %s", result.ServiceName, result.Action, result.Status)
	e.Logger.Info("service %s %s duration: %s", result.ServiceName, result.Action, duration.Round(time.Millisecond))
	if result.BeforeState != "" || result.AfterState != "" {
		e.Logger.Info("service %s state transition: %s -> %s", result.ServiceName, result.BeforeState, result.AfterState)
	}
	for _, warning := range result.Warnings {
		e.Logger.Warn("service %s warning: %s", result.ServiceName, warning)
	}
	for _, errText := range result.Errors {
		e.Logger.Warn("service %s error: %s", result.ServiceName, errText)
	}
}

func shortCommandOutput(output string) string {
	output = strings.Join(strings.Fields(output), " ")
	if len(output) > 500 {
		return output[:500] + "..."
	}
	return output
}

func operation(step CleanupActionType, target string, status app.OperationStatus, message string, errText string) app.OperationResult {
	return app.OperationResult{
		Step:      string(step),
		Target:    target,
		Status:    status,
		Message:   message,
		Error:     errText,
		Timestamp: time.Now(),
	}
}

func serviceOperation(step CleanupActionType, target string, result svc.ServiceActionResult) app.OperationResult {
	errText := strings.Join(result.Errors, "; ")
	message := result.Message
	if len(result.Warnings) > 0 {
		if message != "" {
			message += " "
		}
		message += "Warnings: " + strings.Join(result.Warnings, "; ")
	}
	return operation(step, target, app.OperationStatus(result.Status), message, errText)
}

func serviceRunner(run CommandRunner) svc.CommandRunner {
	return func(name string, args ...string) svc.CommandResult {
		result := run(name, args...)
		return svc.CommandResult{ExitCode: result.ExitCode, Output: result.Output, Err: result.Err}
	}
}

func runCommand(name string, args ...string) CommandResult {
	output, err := exec.Command(name, args...).CombinedOutput()
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

func resultError(result CommandResult) string {
	if strings.TrimSpace(result.Output) != "" {
		return fmt.Sprintf("exit code %d: %s", result.ExitCode, strings.TrimSpace(result.Output))
	}
	if result.Err != nil {
		return fmt.Sprintf("exit code %d: %s", result.ExitCode, result.Err)
	}
	return fmt.Sprintf("exit code %d", result.ExitCode)
}

func currentProcess(run CommandRunner, pid int) (*detector.ProcessState, error) {
	// MVP implementation: uses PowerShell/CIM for process path verification. Keep
	// this isolated so it can later be replaced with native Windows process APIs.
	script := fmt.Sprintf(`$p=Get-CimInstance Win32_Process -Filter "ProcessId = %d" | Select-Object ProcessId,Name,ExecutablePath,CommandLine; if ($null -eq $p) { $null | ConvertTo-Json -Compress } else { $p | ConvertTo-Json -Compress -Depth 3 }`, pid)
	result := run("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	if result.ExitCode != 0 {
		return nil, errors.New(resultError(result))
	}
	processes, err := detector.ParseProcessJSON(result.Output)
	if err != nil {
		return nil, err
	}
	if len(processes) == 0 {
		return nil, nil
	}
	classified := detector.ClassifyProcess(processes[0], detector.MatchOptions{})
	return &classified, nil
}

func (e Executor) waitForProcessExit(pid int) error {
	timeout := e.ProcessTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	poll := e.ProcessPoll
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	for {
		current, err := currentProcess(e.RunCommand, pid)
		if err != nil {
			return err
		}
		if current == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("PID %d still exists after %s", pid, timeout)
		}
		time.Sleep(poll)
	}
}

func plannedProcessActionTrusted(action CleanupAction) bool {
	if action.Type != CleanupActionKillProcess || !action.Safe || action.PID <= 0 {
		return false
	}
	if strings.TrimSpace(action.ProcessName) == "" || strings.TrimSpace(action.ExecutablePath) == "" {
		return false
	}
	switch action.TrustLevel {
	case detector.ProcessTrustNameAndPathMatch, detector.ProcessTrustHiddenWMIExactPath, detector.ProcessTrustTrusted:
		return true
	default:
		return false
	}
}

func validateCurrentProcessMatchesPlan(current detector.ProcessState, action CleanupAction) error {
	if current.PID != action.PID {
		return fmt.Errorf("PID changed: current=%d planned=%d", current.PID, action.PID)
	}
	if !strings.EqualFold(current.Name, action.ProcessName) {
		return fmt.Errorf("process name changed: current=%s planned=%s", current.Name, action.ProcessName)
	}
	if !strings.EqualFold(detector.NormalizeWindowsPath(current.ExecutablePath), detector.NormalizeWindowsPath(action.ExecutablePath)) {
		return fmt.Errorf("process path changed: current=%s planned=%s", current.ExecutablePath, action.ExecutablePath)
	}
	if !detector.TrustedProcessForTermination(current) {
		return fmt.Errorf("current process is not trusted for termination: %s", current.TrustLevel)
	}
	if current.TrustLevel != action.TrustLevel {
		return fmt.Errorf("trust level changed: current=%s planned=%s", current.TrustLevel, action.TrustLevel)
	}
	return nil
}

func parsePID(target string) int {
	re := regexp.MustCompile(`(?i)\bPID\s+(\d+)\b`)
	match := re.FindStringSubmatch(target)
	if len(match) != 2 {
		return 0
	}
	pid, _ := strconv.Atoi(match[1])
	return pid
}

func isAllowedRegistryTarget(target string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(target))
	for _, allowed := range allowedRegistryTargets() {
		if normalized == strings.ToUpper(allowed) {
			return true
		}
	}
	return false
}

func allowedRegistryTargets() []string {
	return []string{
		`HKCU\Software\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`,
		`HKLM\SOFTWARE\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`,
		`HKLM\SOFTWARE\WOW6432Node\Tele Link Soft (TLS) Pte Ltd\TeleLinkSoftHelper`,
		`HKCR\Installer\Features\` + config.MSIPackedCode,
		`HKCR\Installer\Products\` + config.MSIPackedCode,
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + config.MSIProductCode,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\` + config.MSIProductCode,
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Installer\UserData\S-1-5-18\Products\` + config.MSIPackedCode,
	}
}

func splitRegistryTarget(target string) (registry.Key, string, error) {
	parts := strings.SplitN(target, `\`, 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("registry target must include root and path")
	}
	switch strings.ToUpper(parts[0]) {
	case "HKCU":
		return registry.CURRENT_USER, parts[1], nil
	case "HKLM":
		return registry.LOCAL_MACHINE, parts[1], nil
	case "HKCR":
		return registry.CLASSES_ROOT, parts[1], nil
	default:
		return 0, "", fmt.Errorf("unsupported registry root %q", parts[0])
	}
}

func registryKeyExists(root registry.Key, path string) bool {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	key.Close()
	return true
}

func deleteRegistryTree(root registry.Key, path string) error {
	key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE|registry.SET_VALUE)
	if err == nil {
		subkeys, readErr := key.ReadSubKeyNames(-1)
		key.Close()
		if readErr != nil {
			return readErr
		}
		for _, subkey := range subkeys {
			if err := deleteRegistryTree(root, path+`\`+subkey); err != nil {
				return err
			}
		}
	}
	return registry.DeleteKey(root, path)
}

func exportRegistryKey(run CommandRunner, outputDir string, target string) error {
	backupDir := filepath.Join(outputDir, "registry-backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, safeRegistryBackupName(target)+".reg")
	result := run("reg.exe", "export", target, backupPath, "/y")
	if result.ExitCode != 0 {
		return errors.New(resultError(result))
	}
	return nil
}

func safeRegistryBackupName(target string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9_.-]+`)
	return strings.Trim(re.ReplaceAllString(target, "_"), "_")
}
