package config

type Config struct {
	SchemaVersion int             `yaml:"schema_version" json:"schema_version"`
	Profile       string          `yaml:"profile" json:"profile"`
	Reports       ReportsConfig   `yaml:"reports" json:"reports"`
	Installer     InstallerConfig `yaml:"installer" json:"installer"`
	Logging       LoggingConfig   `yaml:"logging" json:"logging"`
	Defender      DefenderConfig  `yaml:"defender" json:"defender"`
	Repair        RepairConfig    `yaml:"repair" json:"repair"`
	Bundle        BundleConfig    `yaml:"bundle" json:"bundle"`
	Wizard        WizardConfig    `yaml:"wizard" json:"wizard"`
}

type ReportsConfig struct {
	Root          string `yaml:"root" json:"root"`
	RetentionDays int    `yaml:"retention_days" json:"retention_days"`
	KeepLast      int    `yaml:"keep_last" json:"keep_last"`
}

type InstallerConfig struct {
	LookupPaths       []string `yaml:"lookup_paths" json:"lookup_paths"`
	PreferredNames    []string `yaml:"preferred_names" json:"preferred_names"`
	RequireValidation bool     `yaml:"require_validation" json:"require_validation"`
	SignaturePolicy   string   `yaml:"signature_policy" json:"signature_policy"`
}

type LoggingConfig struct {
	Level        string `yaml:"level" json:"level"`
	Console      bool   `yaml:"console" json:"console"`
	IncludeDebug bool   `yaml:"include_debug" json:"include_debug"`
}

type DefenderConfig struct {
	EnsureBeforeInstall bool   `yaml:"ensure_before_install" json:"ensure_before_install"`
	MissingPolicy       string `yaml:"missing_policy" json:"missing_policy"`
	QueryTimeoutSeconds int    `yaml:"query_timeout_seconds" json:"query_timeout_seconds"`
}

type RepairConfig struct {
	RequirePreflight             bool `yaml:"require_preflight" json:"require_preflight"`
	RequireRollbackSnapshot      bool `yaml:"require_rollback_snapshot" json:"require_rollback_snapshot"`
	StopOnDefenderEnsureFailure  bool `yaml:"stop_on_defender_ensure_failure" json:"stop_on_defender_ensure_failure"`
	ServiceStopTimeoutSeconds    int  `yaml:"service_stop_timeout_seconds" json:"service_stop_timeout_seconds"`
	ProcessKillTimeoutSeconds    int  `yaml:"process_kill_timeout_seconds" json:"process_kill_timeout_seconds"`
	StopIfDefenderUnavailable    bool `yaml:"stop_if_defender_unavailable" json:"stop_if_defender_unavailable,omitempty"`
	StopIfInstallerMetadataFails bool `yaml:"stop_if_installer_metadata_fails" json:"stop_if_installer_metadata_fails,omitempty"`
}

type BundleConfig struct {
	RedactionEnabled bool `yaml:"redaction_enabled" json:"redaction_enabled"`
	IncludeSystem    bool `yaml:"include_system" json:"include_system"`
	IncludeMSILogs   bool `yaml:"include_msi_logs" json:"include_msi_logs"`
}

type WizardConfig struct {
	CollectBundleOnFailure bool `yaml:"collect_bundle_on_failure" json:"collect_bundle_on_failure"`
	OfferRepair            bool `yaml:"offer_repair" json:"offer_repair"`
	VerifyAfterRepair      bool `yaml:"verify_after_repair" json:"verify_after_repair"`
}

type EffectiveConfig struct {
	Config   Config   `json:"config"`
	Path     string   `json:"path,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type Metadata struct {
	Profile  string   `json:"profile"`
	Path     string   `json:"config_path,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func (e EffectiveConfig) Metadata() Metadata {
	return Metadata{
		Profile:  e.Config.Profile,
		Path:     e.Path,
		Warnings: append([]string(nil), e.Warnings...),
	}
}
