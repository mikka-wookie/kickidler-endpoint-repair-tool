package rollback

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kigrepair/internal/config"
	"kigrepair/internal/detector"
	"kigrepair/internal/winapi"
)

const maxHashBytes int64 = 100 * 1024 * 1024

func CaptureSnapshot(ctx context.Context, input SnapshotInput) (*RollbackInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info := &RollbackInfo{
		SchemaVersion:    SchemaVersion,
		Tool:             ToolName,
		Workflow:         strings.TrimSpace(input.Workflow),
		CreatedAt:        nowString(),
		ReportDir:        input.ReportDir,
		SnapshotOnly:     input.SnapshotOnly,
		RestoreSupported: false,
		RestoreNotes: []string{
			"Automatic rollback is not supported in this version. Use this file as a before-state and change ledger for support escalation. MSI/service/registry restoration must be reviewed manually.",
		},
		HasInvite:     input.HasInvite,
		InstallerPath: input.InstallerPath,
		IsAdmin:       input.IsAdmin,
		Before:        BeforeState{},
	}
	report, _ := input.Detection.(*detector.DetectionReport)
	if report != nil {
		info.Before.DetectionHealth = string(report.Health)
		info.Before.InstallMode = string(report.InstallMode)
		info.Before.InstallRoot = report.InstallRoot
		info.Before.Services = captureServices(report.Services)
		info.Before.Processes = captureProcesses(report.Processes)
		info.Before.Defender = captureDefender(report.Defender, input.RequiredPaths)
	}
	info.Before.Files, info.Before.Directories = capturePaths(input.FileTargets, &info.Warnings)
	info.Before.RegistryKeys = captureRegistry(input.RegistryKeys, report, &info.Warnings)
	info.Before.MSI = captureMSI(info.Before.RegistryKeys)
	info.PlannedChanges = append([]PlannedChange{}, input.PlannedChanges...)
	return info, nil
}

func captureServices(services []detector.ServiceState) []ServiceSnapshot {
	result := make([]ServiceSnapshot, 0, len(services))
	for _, service := range services {
		result = append(result, ServiceSnapshot{
			Name:           service.Name,
			Exists:         service.Exists,
			State:          service.Status,
			StartType:      service.StartType,
			RawImagePath:   firstNonEmpty(service.RawImagePath, service.ImagePath),
			ExecutablePath: firstNonEmpty(service.NormalizedExecutablePath, service.ExecutablePath),
			TrustLevel:     service.TrustLevel,
		})
	}
	return result
}

func captureProcesses(processes []detector.ProcessState) []ProcessSnapshot {
	result := make([]ProcessSnapshot, 0, len(processes))
	for _, process := range processes {
		result = append(result, ProcessSnapshot{
			PID:            process.PID,
			Name:           process.Name,
			ExecutablePath: firstNonEmpty(process.NormalizedExecutablePath, process.ExecutablePath),
			TrustLevel:     process.TrustLevel,
		})
	}
	return result
}

func capturePaths(paths []string, warnings *[]string) ([]FileSnapshot, []DirectorySnapshot) {
	files := make([]FileSnapshot, 0)
	dirs := make([]DirectorySnapshot, 0)
	seen := map[string]bool{}
	for _, target := range paths {
		path := detector.NormalizeWindowsPath(target)
		if path == "" || seen[strings.ToLower(path)] {
			continue
		}
		seen[strings.ToLower(path)] = true
		info, err := os.Lstat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				files = append(files, FileSnapshot{Path: path, Exists: false})
				continue
			}
			files = append(files, FileSnapshot{Path: path, Exists: false, Error: err.Error()})
			*warnings = append(*warnings, "Could not inspect "+path+": "+err.Error())
			continue
		}
		if info.IsDir() {
			dirs = append(dirs, DirectorySnapshot{Path: path, Exists: true})
			continue
		}
		file := FileSnapshot{Path: path, Exists: true, SizeBytes: info.Size()}
		if info.Size() > maxHashBytes {
			file.Error = "hash skipped due to size"
			*warnings = append(*warnings, "Could not hash "+path+": hash skipped due to size")
		} else if sum, err := hashFile(path); err != nil {
			file.Error = err.Error()
			*warnings = append(*warnings, "Could not hash "+path+": "+err.Error())
		} else {
			file.SHA256 = sum
		}
		files = append(files, file)
	}
	return files, dirs
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func captureRegistry(inputKeys []string, report *detector.DetectionReport, warnings *[]string) []RegistrySnapshot {
	keys := append([]string{}, inputKeys...)
	if report != nil {
		for _, key := range report.Registry {
			keys = append(keys, registryPath(key))
		}
	}
	keys = append(keys, knownRegistryTargets()...)
	seen := map[string]bool{}
	result := make([]RegistrySnapshot, 0, len(keys))
	for _, target := range keys {
		target = strings.TrimSpace(target)
		if target == "" || seen[strings.ToUpper(target)] {
			continue
		}
		seen[strings.ToUpper(target)] = true
		snapshot := RegistrySnapshot{Path: target}
		root, path, err := splitRegistryTarget(target)
		if err != nil {
			snapshot.Error = err.Error()
			*warnings = append(*warnings, "Could not query "+target+": "+err.Error())
			result = append(result, snapshot)
			continue
		}
		key := winapi.QueryRegistryKey(context.Background(), root, path, winapi.RegistryViewDefault)
		snapshot.Exists = key.Exists
		snapshot.Accessible = key.Accessible
		if key.Error != "" {
			snapshot.Error = key.Error
			*warnings = append(*warnings, "Could not query "+target+": "+key.Error)
		}
		result = append(result, snapshot)
	}
	return result
}

func captureDefender(state detector.DefenderState, required []string) *DefenderSnapshot {
	status := "unavailable"
	if state.Available {
		status = "available"
	}
	requiredPath := ""
	if len(required) > 0 {
		requiredPath = required[0]
	} else if len(state.RequiredPaths) > 0 {
		requiredPath = state.RequiredPaths[0]
	}
	snapshot := &DefenderSnapshot{
		Status:               status,
		Exclusions:           append([]string{}, state.ExclusionPaths...),
		NormalizedExclusions: append([]string{}, state.NormalizedExclusionPaths...),
		RequiredPath:         requiredPath,
		Covered:              detector.IsPathCoveredByAnyExclusion(requiredPath, state.ExclusionPaths),
	}
	for _, exclusion := range state.ExclusionPaths {
		if detector.IsPathCoveredByExclusion(requiredPath, exclusion) {
			snapshot.CoveredBy = exclusion
			break
		}
	}
	return snapshot
}

func captureMSI(keys []RegistrySnapshot) *MSISnapshot {
	snapshot := &MSISnapshot{ProductCode: config.MSIProductCode, PackedCode: config.MSIPackedCode}
	for _, key := range keys {
		if !key.Exists {
			continue
		}
		upper := strings.ToUpper(key.Path)
		if strings.Contains(upper, strings.ToUpper(config.MSIProductCode)) || strings.Contains(upper, strings.ToUpper(config.MSIPackedCode)) {
			snapshot.Installed = true
			snapshot.UninstallKey = key.Path
			break
		}
	}
	return snapshot
}

func WriteFile(reportDir string, info *RollbackInfo) error {
	if strings.TrimSpace(reportDir) == "" {
		return errors.New("report directory is empty")
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(reportDir, "rollback-info.json"), info)
}

func registryPath(key detector.RegistryState) string {
	if strings.TrimSpace(key.Root) == "" {
		return key.Path
	}
	return key.Root + `\` + key.Path
}

func knownRegistryTargets() []string {
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

func splitRegistryTarget(target string) (string, string, error) {
	parts := strings.SplitN(target, `\`, 2)
	if len(parts) != 2 {
		return "", "", errors.New("registry target must include root and path")
	}
	switch strings.ToUpper(parts[0]) {
	case "HKCU":
		return "HKCU", parts[1], nil
	case "HKLM":
		return "HKLM", parts[1], nil
	case "HKCR":
		return "HKCR", parts[1], nil
	default:
		return "", "", errors.New("unsupported registry root")
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
