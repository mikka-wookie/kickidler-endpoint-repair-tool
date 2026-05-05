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
	b.WriteString(fmt.Sprintf("Reports root: %s\n", e.Config.Reports.Root))
	b.WriteString(fmt.Sprintf("Report retention: %d days, keep last %d\n", e.Config.Reports.RetentionDays, e.Config.Reports.KeepLast))
	b.WriteString(fmt.Sprintf("Logging: level=%s console=%t\n", e.Config.Logging.Level, e.Config.Logging.Console))
	b.WriteString(fmt.Sprintf("Installer validation: required=%t signature_policy=%s\n", e.Config.Installer.RequireValidation, e.Config.Installer.SignaturePolicy))
	b.WriteString(fmt.Sprintf("Defender: ensure_before_install=%t missing_policy=%s timeout=%ds\n", e.Config.Defender.EnsureBeforeInstall, e.Config.Defender.MissingPolicy, e.Config.Defender.QueryTimeoutSeconds))
	b.WriteString(fmt.Sprintf("Repair timeouts: service_stop=%ds process_kill=%ds\n", e.Config.Repair.ServiceStopTimeoutSeconds, e.Config.Repair.ProcessKillTimeoutSeconds))
	b.WriteString(fmt.Sprintf("Bundle: redaction=%t include_system=%t include_msi_logs=%t\n", e.Config.Bundle.RedactionEnabled, e.Config.Bundle.IncludeSystem, e.Config.Bundle.IncludeMSILogs))
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
