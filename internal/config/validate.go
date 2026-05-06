package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidationResult struct {
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func (r ValidationResult) OK() bool {
	return len(r.Errors) == 0
}

func ValidateConfig(cfg Config) ValidationResult {
	var result ValidationResult
	if cfg.SchemaVersion != 1 {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported config schema_version %d", cfg.SchemaVersion))
	}
	if strings.TrimSpace(cfg.Reports.Root) == "" {
		result.Errors = append(result.Errors, "reports.root cannot be empty")
	}
	if cfg.Reports.RetentionDays < 0 {
		result.Errors = append(result.Errors, "reports.retention_days cannot be negative")
	}
	if cfg.Reports.KeepLast < 0 {
		result.Errors = append(result.Errors, "reports.keep_last cannot be negative")
	}
	if !oneOf(cfg.Logging.Level, "debug", "info", "warning", "error") {
		result.Errors = append(result.Errors, "logging.level must be one of debug, info, warning, error")
	}
	if !oneOf(cfg.Logging.Format, "jsonl", "text") {
		result.Errors = append(result.Errors, "logging.format must be one of jsonl, text")
	}
	if !oneOf(cfg.Installer.SignaturePolicy, "warn", "fail", "skip") {
		result.Errors = append(result.Errors, "installer.signature_policy must be one of warn, fail, skip")
	}
	if !oneOf(cfg.Defender.MissingPolicy, "warn", "fail_before_install") {
		result.Errors = append(result.Errors, "defender.missing_policy must be one of warn, fail_before_install")
	}
	if cfg.Defender.QueryTimeoutSeconds < 0 {
		result.Errors = append(result.Errors, "defender.query_timeout_seconds cannot be negative")
	}
	if cfg.Repair.ServiceStopTimeoutSeconds < 0 {
		result.Errors = append(result.Errors, "repair.service_stop_timeout_seconds cannot be negative")
	}
	if cfg.Repair.ProcessKillTimeoutSeconds < 0 {
		result.Errors = append(result.Errors, "repair.process_kill_timeout_seconds cannot be negative")
	}
	if cfg.Bundle.MaxSizeMB < 0 {
		result.Errors = append(result.Errors, "bundle.max_size_mb cannot be negative")
	}
	if _, known := ConfigForProfile(strings.TrimSpace(cfg.Profile)); !known {
		result.Errors = append(result.Errors, "unknown profile name: "+cfg.Profile)
	}
	if !cfg.Repair.RequireAdminForDestructive || !cfg.Safety.RequireAdminForDestructive {
		result.Errors = append(result.Errors, "require_admin_for_destructive cannot be disabled")
	}
	if !cfg.Repair.RequireYesForNonInteractive || !cfg.Safety.RequireYesForNonInteractive {
		result.Errors = append(result.Errors, "require_yes_for_non_interactive cannot be disabled")
	}
	if !cfg.Repair.RequireRollbackSnapshot || !cfg.Safety.RequireRollbackSnapshot {
		result.Errors = append(result.Errors, "require_rollback_snapshot cannot be disabled")
	}
	if !cfg.Cleanup.RequireExactAllowlist || !cfg.Safety.RequireExactAllowlist {
		result.Errors = append(result.Errors, "require_exact_allowlist cannot be disabled")
	}
	if !cfg.Cleanup.RequireRevalidationBeforeMutation || !cfg.Safety.RequireRevalidationBeforeMutation {
		result.Errors = append(result.Errors, "require_revalidation_before_mutation cannot be disabled")
	}

	if !insideDefaultProgramData(cfg.Reports.Root) {
		result.Warnings = append(result.Warnings, "reports.root is outside C:\\ProgramData\\kigrepair")
	}
	if !cfg.Bundle.RedactionEnabled {
		result.Warnings = append(result.Warnings, "bundle.redaction_enabled is disabled")
	}
	if !cfg.Installer.RequireValidation {
		result.Warnings = append(result.Warnings, "installer.require_validation is disabled")
	}
	if !cfg.Defender.EnsureBeforeInstall {
		result.Warnings = append(result.Warnings, "defender.ensure_before_install is disabled")
	}
	for _, path := range cfg.Installer.LookupPaths {
		expanded := ExpandPath(strings.TrimSpace(path))
		if expanded == "" {
			continue
		}
		if _, err := os.Stat(expanded); err != nil {
			result.Warnings = append(result.Warnings, "installer lookup path does not exist: "+path)
		}
	}
	return result
}

func rejectForbiddenKeys(node *yaml.Node) error {
	forbidden := []string{"invite", "password", "token", "access_token", "refresh_token", "secret", "authorization"}
	var walk func(*yaml.Node) error
	walk = func(n *yaml.Node) error {
		if n == nil {
			return nil
		}
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				key := strings.ToLower(strings.TrimSpace(n.Content[i].Value))
				for _, term := range forbidden {
					if strings.Contains(key, term) {
						return fmt.Errorf("config files must not store invite values or secrets. Pass invite via --invite at runtime; forbidden key: %s", n.Content[i].Value)
					}
				}
				if err := walk(n.Content[i+1]); err != nil {
					return err
				}
			}
			return nil
		}
		for _, child := range n.Content {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(node)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if strings.EqualFold(strings.TrimSpace(value), item) {
			return true
		}
	}
	return false
}

func insideDefaultProgramData(path string) bool {
	clean := strings.ToLower(filepath.Clean(ExpandPath(path)))
	root := strings.ToLower(filepath.Clean(`C:\ProgramData\kigrepair`))
	return clean == root || strings.HasPrefix(clean, root+string(os.PathSeparator))
}
