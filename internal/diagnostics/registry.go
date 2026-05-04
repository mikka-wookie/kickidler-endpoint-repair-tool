package diagnostics

import (
	"os/exec"
	"path/filepath"
	"strings"

	"kigrepair/internal/app"
	"kigrepair/internal/detector"
)

type RegistryCollector struct {
	Report detector.DetectionReport
}

func (RegistryCollector) Name() string {
	return "registry"
}

func (c RegistryCollector) Collect(ctx *app.AppContext) app.OperationResult {
	if err := ctx.Reporter.WriteJSON("registry", c.Report.Registry); err != nil {
		return operation("collect.registry", "registry", app.OperationStatusFailed, "Failed to write registry.json", err.Error())
	}
	ctx.Logger.Info("output file created: %s", filepath.Join(ctx.OutputDir, "registry.json"))

	warnings := make([]string, 0)
	if c.Report.IsAdmin {
		for _, key := range c.Report.Registry {
			if !key.Exists {
				continue
			}
			if err := exportRegistryKey(ctx.OutputDir, key); err != nil {
				warnings = append(warnings, err.Error())
			}
		}
	} else {
		warnings = append(warnings, "Registry export skipped because the process is not elevated")
	}
	if len(warnings) > 0 {
		return operation("collect.registry", "registry", app.OperationStatusWarning, "Wrote registry.json with warnings", strings.Join(warnings, "; "))
	}
	return operation("collect.registry", "registry", app.OperationStatusSuccess, "Wrote registry.json", "")
}

func exportRegistryKey(reportDir string, key detector.RegistryState) error {
	exportDir := filepath.Join(reportDir, "registry-export")
	if err := ensureDir(exportDir); err != nil {
		return err
	}
	fullKey := key.Root + `\` + key.Path
	out := filepath.Join(exportDir, safeFilename(key.Root+"_"+key.Path)+".reg")
	output, err := exec.Command("reg.exe", "export", fullKey, out, "/y").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return &collectorError{collector: "registry export", message: fullKey + ": " + message}
	}
	return nil
}
