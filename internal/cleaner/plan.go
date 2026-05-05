package cleaner

import (
	"fmt"
	"os"
	"strings"
	"time"

	"kigrepair/internal/config"
	"kigrepair/internal/detector"
	"kigrepair/internal/safety"
)

type CleanupActionType string

const (
	CleanupActionMSIUninstall      CleanupActionType = "msi_uninstall"
	CleanupActionStopService       CleanupActionType = "stop_service"
	CleanupActionDeleteService     CleanupActionType = "delete_service"
	CleanupActionKillProcess       CleanupActionType = "kill_process"
	CleanupActionDeletePath        CleanupActionType = "delete_path"
	CleanupActionDeleteRegistryKey CleanupActionType = "delete_registry_key"
)

type CleanupAction struct {
	Type           CleanupActionType `json:"type"`
	Target         string            `json:"target"`
	Reason         string            `json:"reason"`
	Safe           bool              `json:"safe"`
	WouldRun       bool              `json:"would_run"`
	PID            int               `json:"pid,omitempty"`
	ProcessName    string            `json:"process_name,omitempty"`
	ExecutablePath string            `json:"executable_path,omitempty"`
	ServiceTrust   string            `json:"service_trust,omitempty"`
	ServiceState   string            `json:"service_state,omitempty"`
	Error          string            `json:"error,omitempty"`
}

type CleanupPlan struct {
	GeneratedAt time.Time       `json:"generated_at"`
	DryRun      bool            `json:"dry_run"`
	Actions     []CleanupAction `json:"actions"`
	Warnings    []string        `json:"warnings"`
	Blockers    []string        `json:"blockers"`
}

type PlanOptions struct {
	DryRun       bool
	CleanupPaths []string
	PathExists   func(string) (bool, error)
	ValidatePath func(string) error
	Now          func() time.Time
}

func BuildPlan(report detector.DetectionReport, opts PlanOptions) CleanupPlan {
	if opts.CleanupPaths == nil {
		opts.CleanupPaths = config.CleanupPaths
	}
	if opts.PathExists == nil {
		opts.PathExists = pathExists
	}
	if opts.ValidatePath == nil {
		opts.ValidatePath = safety.ValidateCleanupPath
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}

	plan := CleanupPlan{
		GeneratedAt: opts.Now(),
		DryRun:      opts.DryRun,
		Actions:     make([]CleanupAction, 0),
		Warnings:    make([]string, 0),
		Blockers:    make([]string, 0),
	}

	if anyMSIRegistryKeyExists(report.Registry) {
		plan.Actions = append(plan.Actions, CleanupAction{
			Type:     CleanupActionMSIUninstall,
			Target:   config.MSIProductCode,
			Reason:   "MSI product or installer registry keys detected",
			Safe:     true,
			WouldRun: opts.DryRun,
		})
	}

	for _, service := range report.Services {
		if !service.Exists {
			continue
		}
		if service.TrustLevel != "trusted" {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("Service %s will not be modified automatically because trust level is %s", service.Name, valueOrUnknown(service.TrustLevel)))
			continue
		}
		plan.Actions = append(plan.Actions,
			CleanupAction{
				Type:           CleanupActionStopService,
				Target:         service.Name,
				Reason:         "Trusted Grabber service detected",
				Safe:           true,
				WouldRun:       opts.DryRun,
				ExecutablePath: service.NormalizedExecutablePath,
				ServiceTrust:   service.TrustLevel,
				ServiceState:   service.Status,
			},
			CleanupAction{
				Type:           CleanupActionDeleteService,
				Target:         service.Name,
				Reason:         "Trusted Grabber service detected",
				Safe:           true,
				WouldRun:       opts.DryRun,
				ExecutablePath: service.NormalizedExecutablePath,
				ServiceTrust:   service.TrustLevel,
				ServiceState:   service.Status,
			},
		)
	}

	for _, process := range report.Processes {
		plan.Actions = append(plan.Actions, CleanupAction{
			Type:           CleanupActionKillProcess,
			Target:         processTarget(process),
			Reason:         "Known Grabber process detected",
			Safe:           true,
			WouldRun:       opts.DryRun,
			PID:            process.PID,
			ProcessName:    process.Name,
			ExecutablePath: process.ExecutablePath,
		})
	}

	for _, configuredPath := range opts.CleanupPaths {
		expandedPath := config.ExpandPath(configuredPath)
		exists, err := opts.PathExists(expandedPath)
		if err != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("Could not check cleanup path %s: %s", expandedPath, err))
			continue
		}
		if !exists {
			continue
		}
		if err := opts.ValidatePath(expandedPath); err != nil {
			plan.Blockers = append(plan.Blockers, fmt.Sprintf("Refused unsafe cleanup path: %s (%s)", expandedPath, err))
			continue
		}
		plan.Actions = append(plan.Actions, CleanupAction{
			Type:     CleanupActionDeletePath,
			Target:   detector.NormalizeWindowsPath(expandedPath),
			Reason:   "Known Grabber cleanup path exists",
			Safe:     true,
			WouldRun: opts.DryRun,
		})
	}

	for _, key := range report.Registry {
		if !key.Exists {
			continue
		}
		plan.Actions = append(plan.Actions, CleanupAction{
			Type:     CleanupActionDeleteRegistryKey,
			Target:   registryTarget(key),
			Reason:   "Known Grabber registry key detected",
			Safe:     true,
			WouldRun: opts.DryRun,
		})
	}

	return plan
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func anyMSIRegistryKeyExists(keys []detector.RegistryState) bool {
	for _, key := range keys {
		if !key.Exists {
			continue
		}
		full := strings.ToUpper(registryTarget(key))
		if strings.Contains(full, strings.ToUpper(config.MSIProductCode)) || strings.Contains(full, strings.ToUpper(config.MSIPackedCode)) {
			return true
		}
	}
	return false
}

func processTarget(process detector.ProcessState) string {
	target := fmt.Sprintf("%s PID %d", process.Name, process.PID)
	if strings.TrimSpace(process.ExecutablePath) != "" {
		target += " " + process.ExecutablePath
	}
	return target
}

func registryTarget(key detector.RegistryState) string {
	if strings.TrimSpace(key.Root) == "" {
		return key.Path
	}
	return key.Root + `\` + key.Path
}
