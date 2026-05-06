package config

func DefaultConfig() Config {
	return Config{
		SchemaVersion: 1,
		Profile:       "standard",
		Reports: ReportsConfig{
			Root:          DefaultReportRoot,
			RetentionDays: 30,
			KeepLast:      10,
		},
		Installer: InstallerConfig{
			LookupPaths: []string{},
			PreferredNames: []string{
				"grabberEM.x64.msi",
				"grabberTT.x64.msi",
				"grabberEM.x32.msi",
				"grabberTT.x32.msi",
				"grabber.msi",
			},
			RequireValidation:         true,
			SignaturePolicy:           "warn",
			RequireSupportedFilename:  true,
			RequireMSIExtension:       true,
			RequireProductCodeMatch:   true,
			AllowMetadataUnavailable:  true,
			AllowSignatureUnavailable: true,
		},
		Logging: LoggingConfig{
			Level:        "info",
			Console:      true,
			IncludeDebug: false,
			Format:       "jsonl",
		},
		Defender: DefenderConfig{
			EnsureBeforeInstall:          true,
			MissingPolicy:                "fail_before_install",
			QueryTimeoutSeconds:          30,
			EnsureEnabled:                true,
			RequireCoverageForInstall:    true,
			AllowAddExclusion:            true,
			TreatPolicyManagedAsBlocking: true,
			TreatUnavailableAsWarning:    false,
		},
		Repair: RepairConfig{
			RequirePreflight:             true,
			RequireRollbackSnapshot:      true,
			StopOnDefenderEnsureFailure:  true,
			ServiceStopTimeoutSeconds:    30,
			ProcessKillTimeoutSeconds:    10,
			StopIfDefenderUnavailable:    false,
			StopIfInstallerMetadataFails: false,
			RequireAdminForDestructive:   true,
			RequireYesForNonInteractive:  true,
			RequireInstallerValidation:   true,
			RequirePreflightReady:        true,
			RequireDefenderBeforeInstall: true,
			AllowRealRepair:              true,
			AllowMSIUninstall:            true,
			AllowServiceDelete:           true,
			AllowProcessTerminate:        true,
			StopOnDefenderFailure:        true,
			StopOnDetectionUnknown:       false,
		},
		Cleanup: CleanupConfig{
			AllowFileDelete:                   true,
			AllowRegistryDelete:               true,
			AllowServiceDelete:                true,
			AllowProcessTerminate:             true,
			AllowHiddenWMICleanup:             true,
			RequireExactAllowlist:             true,
			RequireRevalidationBeforeMutation: true,
			SkipAmbiguousTargets:              true,
		},
		Bundle: BundleConfig{
			RedactionEnabled:         true,
			IncludeSystem:            true,
			IncludeMSILogs:           true,
			IncludeReports:           true,
			IncludeSystemDiagnostics: true,
			IncludeRegistrySnapshot:  true,
			IncludeDefenderSnapshot:  true,
			IncludeProcessSnapshot:   true,
			IncludeServiceSnapshot:   true,
			MaxSizeMB:                50,
		},
		Wizard: WizardConfig{
			CollectBundleOnFailure: true,
			OfferRepair:            true,
			VerifyAfterRepair:      true,
			AllowRealRepairPrompt:  true,
			RequireExactYes:        true,
		},
		Safety: SafetyConfig{
			RequireAdminForDestructive:        true,
			RequireYesForNonInteractive:       true,
			RequireRollbackSnapshot:           true,
			RequireExactAllowlist:             true,
			RequireRevalidationBeforeMutation: true,
		},
	}
}

func ConfigForProfile(profile string) (Config, bool) {
	cfg := DefaultConfig()
	switch profile {
	case "", "standard":
		cfg.Profile = "standard"
		return cfg, true
	case "conservative":
		cfg.Profile = "conservative"
		cfg.Reports.RetentionDays = 60
		cfg.Reports.KeepLast = 20
		cfg.Repair.StopIfDefenderUnavailable = true
		cfg.Repair.StopIfInstallerMetadataFails = true
		cfg.Repair.StopOnDefenderEnsureFailure = true
		cfg.Repair.StopOnDefenderFailure = true
		cfg.Repair.StopOnDetectionUnknown = true
		cfg.Installer.RequireValidation = true
		cfg.Installer.SignaturePolicy = "fail"
		cfg.Defender.MissingPolicy = "fail_before_install"
		cfg.Defender.RequireCoverageForInstall = true
		cfg.Defender.TreatUnavailableAsWarning = false
		cfg.Cleanup.AllowHiddenWMICleanup = false
		cfg.Cleanup.SkipAmbiguousTargets = true
		cfg.Wizard.CollectBundleOnFailure = true
		return cfg, true
	case "diagnostic":
		cfg.Profile = "diagnostic"
		cfg.Reports.RetentionDays = 14
		cfg.Reports.KeepLast = 30
		cfg.Logging.Level = "debug"
		cfg.Logging.IncludeDebug = true
		cfg.Defender.EnsureBeforeInstall = false
		cfg.Defender.AllowAddExclusion = false
		cfg.Defender.RequireCoverageForInstall = false
		cfg.Defender.TreatUnavailableAsWarning = true
		cfg.Wizard.OfferRepair = false
		cfg.Wizard.VerifyAfterRepair = true
		cfg.Wizard.CollectBundleOnFailure = true
		return cfg, true
	default:
		cfg.Profile = profile
		return cfg, false
	}
}

const SampleYAML = `schema_version: 1
profile: standard

reports:
  root: "C:\\ProgramData\\kigrepair\\Reports"
  retention_days: 30
  keep_last: 10

installer:
  lookup_paths:
    - ".\\assets"
  preferred_names:
    - grabberEM.x64.msi
    - grabberTT.x64.msi
    - grabberEM.x32.msi
    - grabberTT.x32.msi
    - grabber.msi
  require_validation: true
  signature_policy: warn

logging:
  level: info
  console: true
  include_debug: false
  format: jsonl

defender:
  ensure_before_install: true
  missing_policy: fail_before_install
  query_timeout_seconds: 30
  ensure_enabled: true
  require_coverage_for_install: true
  allow_add_exclusion: true
  treat_policy_managed_as_blocking: true
  treat_unavailable_as_warning: false

repair:
  require_preflight: true
  require_rollback_snapshot: true
  stop_on_defender_ensure_failure: true
  service_stop_timeout_seconds: 30
  process_kill_timeout_seconds: 10
  require_admin_for_destructive: true
  require_yes_for_non_interactive: true
  require_installer_validation: true
  require_preflight_ready: true
  require_defender_before_install: true
  allow_real_repair: true
  allow_msi_uninstall: true
  allow_service_delete: true
  allow_process_terminate: true
  stop_on_defender_failure: true
  stop_on_detection_unknown: false

cleanup:
  allow_file_delete: true
  allow_registry_delete: true
  allow_service_delete: true
  allow_process_terminate: true
  allow_hidden_wmi_cleanup: true
  require_exact_allowlist: true
  require_revalidation_before_mutation: true
  skip_ambiguous_targets: true

bundle:
  redaction_enabled: true
  include_system: true
  include_msi_logs: true
  include_reports: true
  include_system_diagnostics: true
  include_registry_snapshot: true
  include_defender_snapshot: true
  include_process_snapshot: true
  include_service_snapshot: true
  max_size_mb: 50

wizard:
  collect_bundle_on_failure: true
  offer_repair: true
  verify_after_repair: true
  allow_real_repair_prompt: true
  require_exact_yes: true

safety:
  require_admin_for_destructive: true
  require_yes_for_non_interactive: true
  require_rollback_snapshot: true
  require_exact_allowlist: true
  require_revalidation_before_mutation: true
`
