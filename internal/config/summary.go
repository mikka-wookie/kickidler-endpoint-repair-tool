package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func FormatShow(e EffectiveConfig) string {
	var b strings.Builder
	b.WriteString("Kigrepair Effective Config\n\n")
	b.WriteString("Profile: " + valueOrDash(e.Config.Profile) + "\n")
	b.WriteString("Config path: " + valueOrDash(e.Path) + "\n")
	if e.ProfileOverriddenByCLI {
		b.WriteString("Profile source: CLI override\n")
	}
	b.WriteString(fmt.Sprintf("Reports root: %s\n", e.Config.Reports.Root))
	b.WriteString(fmt.Sprintf("Report retention: %d days, keep last %d\n", e.Config.Reports.RetentionDays, e.Config.Reports.KeepLast))
	b.WriteString(fmt.Sprintf("Logging: level=%s format=%s console=%t\n", e.Config.Logging.Level, e.Config.Logging.Format, e.Config.Logging.Console))
	b.WriteString(fmt.Sprintf("Installer validation: required=%t signature_policy=%s\n", e.Config.Installer.RequireValidation, e.Config.Installer.SignaturePolicy))
	b.WriteString(fmt.Sprintf("Defender: ensure_before_install=%t require_coverage_for_install=%t allow_add_exclusion=%t missing_policy=%s timeout=%ds\n", e.Config.Defender.EnsureBeforeInstall, e.Config.Defender.RequireCoverageForInstall, e.Config.Defender.AllowAddExclusion, e.Config.Defender.MissingPolicy, e.Config.Defender.QueryTimeoutSeconds))
	b.WriteString(fmt.Sprintf("Repair timeouts: service_stop=%ds process_kill=%ds\n", e.Config.Repair.ServiceStopTimeoutSeconds, e.Config.Repair.ProcessKillTimeoutSeconds))
	policy := e.PolicySummary()
	b.WriteString(fmt.Sprintf("Repair policy: allow_real_repair=%t require_rollback=%t require_preflight=%t stop_on_detection_unknown=%t\n", policy.AllowRealRepair, policy.RequireRollbackSnapshot, policy.RequirePreflightReady, policy.StopOnDetectionUnknown))
	b.WriteString(fmt.Sprintf("Cleanup policy: file_delete=%t registry_delete=%t service_delete=%t process_terminate=%t hidden_wmi_cleanup=%t exact_allowlist=%t\n", policy.AllowFileDelete, policy.AllowRegistryDelete, policy.AllowServiceDelete, policy.AllowProcessTerminate, policy.AllowHiddenWMICleanup, policy.RequireExactAllowlist))
	b.WriteString(fmt.Sprintf("Bundle: redaction=%t include_system=%t include_msi_logs=%t max_size_mb=%d\n", e.Config.Bundle.RedactionEnabled, e.Config.Bundle.IncludeSystem, e.Config.Bundle.IncludeMSILogs, e.Config.Bundle.MaxSizeMB))
	b.WriteString(fmt.Sprintf("Wizard: collect_bundle_on_failure=%t offer_repair=%t verify_after_repair=%t\n", e.Config.Wizard.CollectBundleOnFailure, e.Config.Wizard.OfferRepair, e.Config.Wizard.VerifyAfterRepair))
	if len(e.Config.Installer.LookupPaths) > 0 {
		b.WriteString("\nInstaller lookup paths:\n")
		for _, path := range e.Config.Installer.LookupPaths {
			b.WriteString("- " + path + "\n")
		}
	}
	if len(e.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range e.Warnings {
			b.WriteString("- " + warning + "\n")
		}
	}
	return b.String()
}

func FormatValidation(result ValidationResult) string {
	var b strings.Builder
	if result.OK() {
		b.WriteString("Config validation: success\n")
	} else {
		b.WriteString("Config validation: failed\n")
	}
	if len(result.Errors) > 0 {
		b.WriteString("\nErrors:\n")
		for _, errText := range result.Errors {
			b.WriteString("- " + errText + "\n")
		}
	}
	if len(result.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range result.Warnings {
			b.WriteString("- " + warning + "\n")
		}
	}
	return b.String()
}

func MarshalJSON(e EffectiveConfig) ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

func MarshalYAMLConfig(cfg Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
