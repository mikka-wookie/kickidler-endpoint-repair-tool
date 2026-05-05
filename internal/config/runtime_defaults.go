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
			RequireValidation: true,
			SignaturePolicy:   "warn",
		},
		Logging: LoggingConfig{
			Level:        "info",
			Console:      true,
			IncludeDebug: false,
		},
		Defender: DefenderConfig{
			EnsureBeforeInstall: true,
			MissingPolicy:       "fail_before_install",
			QueryTimeoutSeconds: 30,
		},
		Repair: RepairConfig{
			RequirePreflight:             true,
			RequireRollbackSnapshot:      true,
			StopOnDefenderEnsureFailure:  true,
			ServiceStopTimeoutSeconds:    30,
			ProcessKillTimeoutSeconds:    10,
			StopIfDefenderUnavailable:    false,
			StopIfInstallerMetadataFails: false,
		},
		Bundle: BundleConfig{
			RedactionEnabled: true,
			IncludeSystem:    true,
			IncludeMSILogs:   true,
		},
		Wizard: WizardConfig{
			CollectBundleOnFailure: true,
			OfferRepair:            true,
			VerifyAfterRepair:      true,
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
		cfg.Repair.StopIfDefenderUnavailable = true
		cfg.Repair.StopIfInstallerMetadataFails = true
		cfg.Repair.StopOnDefenderEnsureFailure = true
		cfg.Installer.RequireValidation = true
		cfg.Installer.SignaturePolicy = "fail"
		cfg.Defender.MissingPolicy = "fail_before_install"
		cfg.Wizard.CollectBundleOnFailure = true
		return cfg, true
	case "diagnostic":
		cfg.Profile = "diagnostic"
		cfg.Defender.EnsureBeforeInstall = false
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

defender:
  ensure_before_install: true
  missing_policy: fail_before_install
  query_timeout_seconds: 30

repair:
  require_preflight: true
  require_rollback_snapshot: true
  stop_on_defender_ensure_failure: true
  service_stop_timeout_seconds: 30
  process_kill_timeout_seconds: 10

bundle:
  redaction_enabled: true
  include_system: true
  include_msi_logs: true

wizard:
  collect_bundle_on_failure: true
  offer_repair: true
  verify_after_repair: true
`
