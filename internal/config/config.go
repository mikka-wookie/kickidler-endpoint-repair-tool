package config

type Config struct {
	SchemaVersion int             `yaml:"schema_version" json:"schema_version"`
	Profile       string          `yaml:"profile" json:"profile"`
	Reports       ReportsConfig   `yaml:"reports" json:"reports"`
	Installer     InstallerConfig `yaml:"installer" json:"installer"`
	Logging       LoggingConfig   `yaml:"logging" json:"logging"`
	Defender      DefenderConfig  `yaml:"defender" json:"defender"`
	Repair        RepairConfig    `yaml:"repair" json:"repair"`
	Cleanup       CleanupConfig   `yaml:"cleanup" json:"cleanup"`
	Bundle        BundleConfig    `yaml:"bundle" json:"bundle"`
	Wizard        WizardConfig    `yaml:"wizard" json:"wizard"`
	Safety        SafetyConfig    `yaml:"safety" json:"safety"`
}

type ReportsConfig struct {
	Root          string `yaml:"root" json:"root"`
	RetentionDays int    `yaml:"retention_days" json:"retention_days"`
	KeepLast      int    `yaml:"keep_last" json:"keep_last"`
}

type InstallerConfig struct {
	LookupPaths               []string `yaml:"lookup_paths" json:"lookup_paths"`
	PreferredNames            []string `yaml:"preferred_names" json:"preferred_names"`
	RequireValidation         bool     `yaml:"require_validation" json:"require_validation"`
	SignaturePolicy           string   `yaml:"signature_policy" json:"signature_policy"`
	RequireSupportedFilename  bool     `yaml:"require_supported_filename" json:"require_supported_filename"`
	RequireMSIExtension       bool     `yaml:"require_msi_extension" json:"require_msi_extension"`
	RequireProductCodeMatch   bool     `yaml:"require_product_code_match" json:"require_product_code_match"`
	AllowMetadataUnavailable  bool     `yaml:"allow_metadata_unavailable" json:"allow_metadata_unavailable"`
	AllowSignatureUnavailable bool     `yaml:"allow_signature_unavailable" json:"allow_signature_unavailable"`
}

type LoggingConfig struct {
	Level        string `yaml:"level" json:"level"`
	Console      bool   `yaml:"console" json:"console"`
	IncludeDebug bool   `yaml:"include_debug" json:"include_debug"`
	Format       string `yaml:"format" json:"format"`
}

type DefenderConfig struct {
	EnsureBeforeInstall          bool   `yaml:"ensure_before_install" json:"ensure_before_install"`
	MissingPolicy                string `yaml:"missing_policy" json:"missing_policy"`
	QueryTimeoutSeconds          int    `yaml:"query_timeout_seconds" json:"query_timeout_seconds"`
	EnsureEnabled                bool   `yaml:"ensure_enabled" json:"ensure_enabled"`
	RequireCoverageForInstall    bool   `yaml:"require_coverage_for_install" json:"require_coverage_for_install"`
	AllowAddExclusion            bool   `yaml:"allow_add_exclusion" json:"allow_add_exclusion"`
	TreatPolicyManagedAsBlocking bool   `yaml:"treat_policy_managed_as_blocking" json:"treat_policy_managed_as_blocking"`
	TreatUnavailableAsWarning    bool   `yaml:"treat_unavailable_as_warning" json:"treat_unavailable_as_warning"`
}

type RepairConfig struct {
	RequirePreflight             bool `yaml:"require_preflight" json:"require_preflight"`
	RequireRollbackSnapshot      bool `yaml:"require_rollback_snapshot" json:"require_rollback_snapshot"`
	StopOnDefenderEnsureFailure  bool `yaml:"stop_on_defender_ensure_failure" json:"stop_on_defender_ensure_failure"`
	ServiceStopTimeoutSeconds    int  `yaml:"service_stop_timeout_seconds" json:"service_stop_timeout_seconds"`
	ProcessKillTimeoutSeconds    int  `yaml:"process_kill_timeout_seconds" json:"process_kill_timeout_seconds"`
	StopIfDefenderUnavailable    bool `yaml:"stop_if_defender_unavailable" json:"stop_if_defender_unavailable,omitempty"`
	StopIfInstallerMetadataFails bool `yaml:"stop_if_installer_metadata_fails" json:"stop_if_installer_metadata_fails,omitempty"`
	RequireAdminForDestructive   bool `yaml:"require_admin_for_destructive" json:"require_admin_for_destructive"`
	RequireYesForNonInteractive  bool `yaml:"require_yes_for_non_interactive" json:"require_yes_for_non_interactive"`
	RequireInstallerValidation   bool `yaml:"require_installer_validation" json:"require_installer_validation"`
	RequirePreflightReady        bool `yaml:"require_preflight_ready" json:"require_preflight_ready"`
	RequireDefenderBeforeInstall bool `yaml:"require_defender_before_install" json:"require_defender_before_install"`
	AllowRealRepair              bool `yaml:"allow_real_repair" json:"allow_real_repair"`
	AllowMSIUninstall            bool `yaml:"allow_msi_uninstall" json:"allow_msi_uninstall"`
	AllowServiceDelete           bool `yaml:"allow_service_delete" json:"allow_service_delete"`
	AllowProcessTerminate        bool `yaml:"allow_process_terminate" json:"allow_process_terminate"`
	StopOnDefenderFailure        bool `yaml:"stop_on_defender_failure" json:"stop_on_defender_failure"`
	StopOnDetectionUnknown       bool `yaml:"stop_on_detection_unknown" json:"stop_on_detection_unknown"`
}

type CleanupConfig struct {
	AllowFileDelete                   bool `yaml:"allow_file_delete" json:"allow_file_delete"`
	AllowRegistryDelete               bool `yaml:"allow_registry_delete" json:"allow_registry_delete"`
	AllowServiceDelete                bool `yaml:"allow_service_delete" json:"allow_service_delete"`
	AllowProcessTerminate             bool `yaml:"allow_process_terminate" json:"allow_process_terminate"`
	AllowHiddenWMICleanup             bool `yaml:"allow_hidden_wmi_cleanup" json:"allow_hidden_wmi_cleanup"`
	RequireExactAllowlist             bool `yaml:"require_exact_allowlist" json:"require_exact_allowlist"`
	RequireRevalidationBeforeMutation bool `yaml:"require_revalidation_before_mutation" json:"require_revalidation_before_mutation"`
	SkipAmbiguousTargets              bool `yaml:"skip_ambiguous_targets" json:"skip_ambiguous_targets"`
}

type BundleConfig struct {
	RedactionEnabled         bool `yaml:"redaction_enabled" json:"redaction_enabled"`
	IncludeSystem            bool `yaml:"include_system" json:"include_system"`
	IncludeMSILogs           bool `yaml:"include_msi_logs" json:"include_msi_logs"`
	IncludeReports           bool `yaml:"include_reports" json:"include_reports"`
	IncludeSystemDiagnostics bool `yaml:"include_system_diagnostics" json:"include_system_diagnostics"`
	IncludeRegistrySnapshot  bool `yaml:"include_registry_snapshot" json:"include_registry_snapshot"`
	IncludeDefenderSnapshot  bool `yaml:"include_defender_snapshot" json:"include_defender_snapshot"`
	IncludeProcessSnapshot   bool `yaml:"include_process_snapshot" json:"include_process_snapshot"`
	IncludeServiceSnapshot   bool `yaml:"include_service_snapshot" json:"include_service_snapshot"`
	MaxSizeMB                int  `yaml:"max_size_mb" json:"max_size_mb"`
}

type WizardConfig struct {
	CollectBundleOnFailure bool `yaml:"collect_bundle_on_failure" json:"collect_bundle_on_failure"`
	OfferRepair            bool `yaml:"offer_repair" json:"offer_repair"`
	VerifyAfterRepair      bool `yaml:"verify_after_repair" json:"verify_after_repair"`
	AllowRealRepairPrompt  bool `yaml:"allow_real_repair_prompt" json:"allow_real_repair_prompt"`
	RequireExactYes        bool `yaml:"require_exact_yes" json:"require_exact_yes"`
}

type SafetyConfig struct {
	RequireAdminForDestructive        bool `yaml:"require_admin_for_destructive" json:"require_admin_for_destructive"`
	RequireYesForNonInteractive       bool `yaml:"require_yes_for_non_interactive" json:"require_yes_for_non_interactive"`
	RequireRollbackSnapshot           bool `yaml:"require_rollback_snapshot" json:"require_rollback_snapshot"`
	RequireExactAllowlist             bool `yaml:"require_exact_allowlist" json:"require_exact_allowlist"`
	RequireRevalidationBeforeMutation bool `yaml:"require_revalidation_before_mutation" json:"require_revalidation_before_mutation"`
}

type PolicySource struct {
	ConfigPath             string `json:"config_path,omitempty"`
	ProfileOverriddenByCLI bool   `json:"profile_overridden_by_cli,omitempty"`
}

type PolicySummary struct {
	Profile                           string       `json:"profile"`
	Source                            PolicySource `json:"source"`
	RequireRollbackSnapshot           bool         `json:"require_rollback_snapshot"`
	RequireAdminForDestructive        bool         `json:"require_admin_for_destructive"`
	RequireYesForNonInteractive       bool         `json:"require_yes_for_non_interactive"`
	RequireDefenderBeforeInstall      bool         `json:"require_defender_before_install"`
	RequireInstallerValidation        bool         `json:"require_installer_validation"`
	RequirePreflightReady             bool         `json:"require_preflight_ready"`
	AllowRealRepair                   bool         `json:"allow_real_repair"`
	AllowMSIUninstall                 bool         `json:"allow_msi_uninstall"`
	AllowServiceDelete                bool         `json:"allow_service_delete"`
	AllowProcessTerminate             bool         `json:"allow_process_terminate"`
	AllowFileDelete                   bool         `json:"allow_file_delete"`
	AllowRegistryDelete               bool         `json:"allow_registry_delete"`
	AllowHiddenWMICleanup             bool         `json:"allow_hidden_wmi_cleanup"`
	RequireExactAllowlist             bool         `json:"require_exact_allowlist"`
	RequireRevalidationBeforeMutation bool         `json:"require_revalidation_before_mutation"`
	SkipAmbiguousTargets              bool         `json:"skip_ambiguous_targets"`
	StopOnDetectionUnknown            bool         `json:"stop_on_detection_unknown"`
	DefenderRequireCoverageForInstall bool         `json:"defender_require_coverage_for_install"`
	DefenderAllowAddExclusion         bool         `json:"defender_allow_add_exclusion"`
	DefenderTreatUnavailableAsWarning bool         `json:"defender_treat_unavailable_as_warning"`
	ReportRetentionDays               int          `json:"report_retention_days"`
	ReportKeepLast                    int          `json:"report_keep_last"`
	LoggingLevel                      string       `json:"logging_level"`
}

type EffectiveConfig struct {
	Config                 Config   `json:"config"`
	Path                   string   `json:"path,omitempty"`
	Warnings               []string `json:"warnings,omitempty"`
	ProfileOverriddenByCLI bool     `json:"profile_overridden_by_cli,omitempty"`
}

type Metadata struct {
	Profile  string        `json:"profile"`
	Path     string        `json:"config_path,omitempty"`
	Warnings []string      `json:"warnings,omitempty"`
	Policy   PolicySummary `json:"policy"`
}

func (e EffectiveConfig) Metadata() Metadata {
	return Metadata{
		Profile:  e.Config.Profile,
		Path:     e.Path,
		Warnings: append([]string(nil), e.Warnings...),
		Policy:   e.PolicySummary(),
	}
}

func (e EffectiveConfig) PolicySummary() PolicySummary {
	return PolicySummary{
		Profile:                           e.Config.Profile,
		Source:                            PolicySource{ConfigPath: e.Path, ProfileOverriddenByCLI: e.ProfileOverriddenByCLI},
		RequireRollbackSnapshot:           e.Config.Repair.RequireRollbackSnapshot && e.Config.Safety.RequireRollbackSnapshot,
		RequireAdminForDestructive:        e.Config.Repair.RequireAdminForDestructive && e.Config.Safety.RequireAdminForDestructive,
		RequireYesForNonInteractive:       e.Config.Repair.RequireYesForNonInteractive && e.Config.Safety.RequireYesForNonInteractive,
		RequireDefenderBeforeInstall:      e.Config.Repair.RequireDefenderBeforeInstall,
		RequireInstallerValidation:        e.Config.Repair.RequireInstallerValidation,
		RequirePreflightReady:             e.Config.Repair.RequirePreflightReady,
		AllowRealRepair:                   e.Config.Repair.AllowRealRepair,
		AllowMSIUninstall:                 e.Config.Repair.AllowMSIUninstall,
		AllowServiceDelete:                e.Config.Repair.AllowServiceDelete && e.Config.Cleanup.AllowServiceDelete,
		AllowProcessTerminate:             e.Config.Repair.AllowProcessTerminate && e.Config.Cleanup.AllowProcessTerminate,
		AllowFileDelete:                   e.Config.Cleanup.AllowFileDelete,
		AllowRegistryDelete:               e.Config.Cleanup.AllowRegistryDelete,
		AllowHiddenWMICleanup:             e.Config.Cleanup.AllowHiddenWMICleanup,
		RequireExactAllowlist:             e.Config.Cleanup.RequireExactAllowlist && e.Config.Safety.RequireExactAllowlist,
		RequireRevalidationBeforeMutation: e.Config.Cleanup.RequireRevalidationBeforeMutation && e.Config.Safety.RequireRevalidationBeforeMutation,
		SkipAmbiguousTargets:              e.Config.Cleanup.SkipAmbiguousTargets,
		StopOnDetectionUnknown:            e.Config.Repair.StopOnDetectionUnknown,
		DefenderRequireCoverageForInstall: e.Config.Defender.RequireCoverageForInstall,
		DefenderAllowAddExclusion:         e.Config.Defender.AllowAddExclusion,
		DefenderTreatUnavailableAsWarning: e.Config.Defender.TreatUnavailableAsWarning,
		ReportRetentionDays:               e.Config.Reports.RetentionDays,
		ReportKeepLast:                    e.Config.Reports.KeepLast,
		LoggingLevel:                      e.Config.Logging.Level,
	}
}
