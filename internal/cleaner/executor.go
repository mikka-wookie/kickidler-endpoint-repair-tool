package cleaner

import (
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
	// MVP implementation: uses sc.exe. Keep this isolated so it can later be
	// replaced with native Windows service APIs.
	if !serviceExists(e.RunCommand, action.Target) {
		return operation(action.Type, action.Target, app.OperationStatusSkipped, "Service does not exist", "")
	}
	state := serviceState(e.RunCommand, action.Target)
	if strings.EqualFold(state, "stopped") {
		return operation(action.Type, action.Target, app.OperationStatusSkipped, "Service is already stopped", "")
	}
	if e.Logger != nil {
		e.Logger.Info("executing command: sc.exe stop %s", action.Target)
	}
	started := time.Now()
	result := e.RunCommand("sc.exe", "stop", action.Target)
	e.logCommandResult("sc.exe", result, time.Since(started))
	if result.ExitCode != 0 && !strings.Contains(result.Output, "1062") {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Service stop failed", resultError(result))
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if strings.EqualFold(serviceState(e.RunCommand, action.Target), "stopped") {
			return operation(action.Type, action.Target, app.OperationStatusSuccess, "Service stopped", "")
		}
		time.Sleep(500 * time.Millisecond)
	}
	return operation(action.Type, action.Target, app.OperationStatusFailed, "Timed out waiting for service to stop", "")
}

func (e Executor) deleteService(action CleanupAction) app.OperationResult {
	// MVP implementation: uses sc.exe. Keep this isolated so it can later be
	// replaced with native Windows service APIs.
	if !serviceExists(e.RunCommand, action.Target) {
		return operation(action.Type, action.Target, app.OperationStatusSkipped, "Service does not exist", "")
	}
	if e.Logger != nil {
		e.Logger.Info("executing command: sc.exe delete %s", action.Target)
	}
	started := time.Now()
	result := e.RunCommand("sc.exe", "delete", action.Target)
	e.logCommandResult("sc.exe", result, time.Since(started))
	if result.ExitCode != 0 {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Service deletion failed", resultError(result))
	}
	return operation(action.Type, action.Target, app.OperationStatusSuccess, "Service deleted", "")
}

func (e Executor) killProcess(action CleanupAction) app.OperationResult {
	// MVP implementation: uses taskkill.exe by PID. Keep this isolated so it can
	// later be replaced with native Windows process APIs.
	pid := action.PID
	if pid == 0 {
		pid = parsePID(action.Target)
	}
	if pid <= 0 {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Cleanup plan does not contain a valid PID", "")
	}
	if !processExists(e.RunCommand, pid) {
		return operation(action.Type, action.Target, app.OperationStatusSkipped, "Process does not exist", "")
	}
	if action.ExecutablePath != "" {
		currentPath := currentProcessExecutablePath(e.RunCommand, pid)
		if currentPath != "" && !strings.EqualFold(detector.NormalizeWindowsPath(currentPath), detector.NormalizeWindowsPath(action.ExecutablePath)) {
			return operation(action.Type, action.Target, app.OperationStatusFailed, "Process executable path no longer matches cleanup plan", currentPath)
		}
		if !isKnownGrabberProcessPath(action.ExecutablePath) {
			return operation(action.Type, action.Target, app.OperationStatusFailed, "Process executable path is not an allowed Grabber path", action.ExecutablePath)
		}
	}
	if e.Logger != nil {
		e.Logger.Info("executing command: taskkill /PID %d /F", pid)
	}
	started := time.Now()
	result := e.RunCommand("taskkill.exe", "/PID", strconv.Itoa(pid), "/F")
	e.logCommandResult("taskkill.exe", result, time.Since(started))
	if result.ExitCode != 0 {
		return operation(action.Type, action.Target, app.OperationStatusFailed, "Process kill failed", resultError(result))
	}
	return operation(action.Type, action.Target, app.OperationStatusSuccess, "Process killed", "")
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

func serviceExists(run CommandRunner, name string) bool {
	// MVP implementation: uses sc.exe query. Keep this isolated so it can later
	// be replaced with native Windows service APIs.
	result := run("sc.exe", "query", name)
	return result.ExitCode == 0
}

func serviceState(run CommandRunner, name string) string {
	// MVP implementation: uses sc.exe query. Keep this isolated so it can later
	// be replaced with native Windows service APIs.
	result := run("sc.exe", "query", name)
	re := regexp.MustCompile(`(?m)^\s*STATE\s*:\s*\d+\s+([A-Z_]+)`)
	if match := re.FindStringSubmatch(result.Output); len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return ""
}

func processExists(run CommandRunner, pid int) bool {
	// MVP implementation: uses tasklist.exe for PID existence checks. Keep this
	// isolated so it can later be replaced with native Windows process APIs.
	result := run("tasklist.exe", "/FI", "PID eq "+strconv.Itoa(pid), "/NH")
	if result.ExitCode != 0 {
		return false
	}
	return strings.Contains(result.Output, strconv.Itoa(pid))
}

func currentProcessExecutablePath(run CommandRunner, pid int) string {
	// MVP implementation: uses PowerShell/CIM for process path verification. Keep
	// this isolated so it can later be replaced with native Windows process APIs.
	script := fmt.Sprintf(`$p=Get-CimInstance Win32_Process -Filter "ProcessId = %d"; if ($null -ne $p) { $p.ExecutablePath }`, pid)
	result := run("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	if result.ExitCode != 0 {
		return ""
	}
	return strings.TrimSpace(result.Output)
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

func isKnownGrabberProcessPath(path string) bool {
	normalized := strings.ToLower(detector.NormalizeWindowsPath(path))
	if normalized == "" {
		return true
	}
	for _, cleanupPath := range config.ExpandedCleanupPaths() {
		root := strings.ToLower(detector.NormalizeWindowsPath(cleanupPath))
		if normalized == root || strings.HasPrefix(normalized, root+`\`) {
			return true
		}
	}
	for _, wmiPath := range config.KnownWMIExecutablePaths {
		if normalized == strings.ToLower(detector.NormalizeWindowsPath(config.ExpandPath(wmiPath))) {
			return true
		}
	}
	return false
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
