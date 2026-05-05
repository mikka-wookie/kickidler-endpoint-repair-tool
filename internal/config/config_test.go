package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadNoConfigUsesSafeDefaults(t *testing.T) {
	effective, err := Load(LoadOptions{
		ExeDir:         t.TempDir(),
		ProgramDataDir: t.TempDir(),
		WorkDir:        t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if effective.Config.Profile != "standard" {
		t.Fatalf("Profile = %q, want standard", effective.Config.Profile)
	}
	if effective.Config.Reports.Root != DefaultReportRoot {
		t.Fatalf("Reports.Root = %q, want %q", effective.Config.Reports.Root, DefaultReportRoot)
	}
	if !effective.Config.Bundle.RedactionEnabled {
		t.Fatal("redaction default must be enabled")
	}
	if !effective.Config.Installer.RequireValidation {
		t.Fatal("installer validation default must be enabled")
	}
}

func TestLoadExplicitConfig(t *testing.T) {
	path := writeConfig(t, t.TempDir(), `schema_version: 1
profile: standard
reports:
  root: "D:\\Reports"
`)
	effective, err := Load(LoadOptions{ExplicitPath: path})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if effective.Path != path {
		t.Fatalf("Path = %q, want %q", effective.Path, path)
	}
	if effective.Config.Reports.Root != `D:\Reports` {
		t.Fatalf("Reports.Root = %q", effective.Config.Reports.Root)
	}
}

func TestResolveConfigLookupOrder(t *testing.T) {
	exeDir := t.TempDir()
	programData := t.TempDir()
	workDir := t.TempDir()
	explicit := writeConfig(t, t.TempDir(), `schema_version: 1`)
	writeConfig(t, exeDir, `schema_version: 1
profile: conservative
`)
	writeConfig(t, programData, `schema_version: 1
profile: diagnostic
`)
	writeConfig(t, workDir, `schema_version: 1
profile: standard
`)

	got, explicitUsed, err := ResolveConfigPath(LoadOptions{ExplicitPath: explicit, ExeDir: exeDir, ProgramDataDir: programData, WorkDir: workDir})
	if err != nil {
		t.Fatalf("ResolveConfigPath explicit error = %v", err)
	}
	if !explicitUsed || got != explicit {
		t.Fatalf("explicit path got %q explicit=%t", got, explicitUsed)
	}

	got, explicitUsed, err = ResolveConfigPath(LoadOptions{ExeDir: exeDir, ProgramDataDir: programData, WorkDir: workDir})
	if err != nil {
		t.Fatalf("ResolveConfigPath lookup error = %v", err)
	}
	want := filepath.Join(exeDir, ConfigFileName)
	if explicitUsed || got != want {
		t.Fatalf("lookup got %q explicit=%t, want %q false", got, explicitUsed, want)
	}
}

func TestUnknownExplicitConfigReturnsError(t *testing.T) {
	_, err := Load(LoadOptions{ExplicitPath: filepath.Join(t.TempDir(), "missing.yaml")})
	if err == nil || !strings.Contains(err.Error(), "config file not found") {
		t.Fatalf("Load() error = %v, want config file not found", err)
	}
}

func TestProfiles(t *testing.T) {
	standard, ok := ConfigForProfile("standard")
	if !ok || standard.Profile != "standard" || !standard.Defender.EnsureBeforeInstall {
		t.Fatalf("standard profile is not safe: %+v ok=%t", standard, ok)
	}
	conservative, ok := ConfigForProfile("conservative")
	if !ok || !conservative.Repair.StopIfDefenderUnavailable || conservative.Installer.SignaturePolicy != "fail" {
		t.Fatalf("conservative profile not stricter: %+v ok=%t", conservative, ok)
	}
	diagnostic, ok := ConfigForProfile("diagnostic")
	if !ok || diagnostic.Wizard.OfferRepair {
		t.Fatalf("diagnostic profile should not offer repair: %+v ok=%t", diagnostic, ok)
	}
}

func TestValidationErrorsAndWarnings(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		errPart string
		warn    string
	}{
		{name: "negative retention", mutate: func(c *Config) { c.Reports.RetentionDays = -1 }, errPart: "retention_days"},
		{name: "bad logging", mutate: func(c *Config) { c.Logging.Level = "trace" }, errPart: "logging.level"},
		{name: "bad signature", mutate: func(c *Config) { c.Installer.SignaturePolicy = "maybe" }, errPart: "signature_policy"},
		{name: "bad defender", mutate: func(c *Config) { c.Defender.MissingPolicy = "ignore" }, errPart: "missing_policy"},
		{name: "redaction warning", mutate: func(c *Config) { c.Bundle.RedactionEnabled = false }, warn: "redaction"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(&cfg)
			result := ValidateConfig(cfg)
			if tt.errPart != "" && !containsText(result.Errors, tt.errPart) {
				t.Fatalf("Errors = %v, want %q", result.Errors, tt.errPart)
			}
			if tt.warn != "" && !containsText(result.Warnings, tt.warn) {
				t.Fatalf("Warnings = %v, want %q", result.Warnings, tt.warn)
			}
		})
	}
}

func TestForbiddenKeysFail(t *testing.T) {
	for _, key := range []string{"invite", "token", "access_token", "refresh_token", "password", "secret", "authorization"} {
		t.Run(key, func(t *testing.T) {
			_, err := Parse([]byte("schema_version: 1\n" + key + ": raw\n"))
			if err == nil || !strings.Contains(err.Error(), "must not store invite values or secrets") {
				t.Fatalf("Parse() error = %v", err)
			}
		})
	}
}

func TestSampleConfigIsValidYAML(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(SampleYAML), &cfg); err != nil {
		t.Fatalf("sample yaml invalid: %v", err)
	}
	effective, err := Parse([]byte(SampleYAML))
	if err != nil {
		t.Fatalf("Parse(sample) error = %v", err)
	}
	result := ValidateConfig(effective.Config)
	if len(result.Errors) > 0 {
		t.Fatalf("sample validation errors = %v", result.Errors)
	}
}

func TestConfigShowDoesNotPrintInvite(t *testing.T) {
	effective := EffectiveConfig{Config: DefaultConfig()}
	output := strings.ToLower(FormatShow(effective))
	if strings.Contains(output, "raw-secret-invite") {
		t.Fatal("config show printed a secret")
	}
}

func writeConfig(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func containsText(values []string, part string) bool {
	for _, value := range values {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
