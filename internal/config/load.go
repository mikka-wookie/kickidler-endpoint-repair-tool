package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = "kigrepair.yaml"

type LoadOptions struct {
	ExplicitPath   string
	ExeDir         string
	ProgramDataDir string
	WorkDir        string
}

func Load(opts LoadOptions) (EffectiveConfig, error) {
	path, explicit, err := ResolveConfigPath(opts)
	if err != nil {
		return EffectiveConfig{}, err
	}
	if path == "" {
		cfg := DefaultConfig()
		warnings := ValidateConfig(cfg)
		return EffectiveConfig{Config: cfg, Warnings: warnings.Warnings}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return EffectiveConfig{}, fmt.Errorf("read config %q: %w", path, err)
	}
	effective, err := Parse(data)
	if err != nil {
		return EffectiveConfig{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	effective.Path = path
	validation := ValidateConfig(effective.Config)
	effective.Warnings = append(effective.Warnings, validation.Warnings...)
	if len(validation.Errors) > 0 {
		return effective, errors.New(strings.Join(validation.Errors, "; "))
	}
	if explicit && effective.Path == "" {
		effective.Path = path
	}
	return effective, nil
}

func Parse(data []byte) (EffectiveConfig, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return EffectiveConfig{}, err
	}
	if err := rejectForbiddenKeys(&node); err != nil {
		return EffectiveConfig{}, err
	}

	var header struct {
		SchemaVersion int    `yaml:"schema_version"`
		Profile       string `yaml:"profile"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return EffectiveConfig{}, err
	}
	if header.SchemaVersion == 0 {
		header.SchemaVersion = 1
	}
	if header.SchemaVersion != 1 {
		return EffectiveConfig{}, fmt.Errorf("unsupported config schema_version %d", header.SchemaVersion)
	}

	profile := strings.TrimSpace(header.Profile)
	if profile == "" {
		profile = "standard"
	}
	cfg, knownProfile := ConfigForProfile(profile)
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return EffectiveConfig{}, err
	}
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = 1
	}
	if strings.TrimSpace(cfg.Profile) == "" {
		cfg.Profile = profile
	}

	warnings := []string{}
	if !knownProfile {
		warnings = append(warnings, "unknown profile name: "+profile)
	}
	return EffectiveConfig{Config: cfg, Warnings: warnings}, nil
}

func ResolveConfigPath(opts LoadOptions) (string, bool, error) {
	if strings.TrimSpace(opts.ExplicitPath) != "" {
		path, err := filepath.Abs(ExpandPath(strings.TrimSpace(opts.ExplicitPath)))
		if err != nil {
			return "", true, err
		}
		if _, err := os.Stat(path); err != nil {
			return "", true, fmt.Errorf("config file not found: %s", path)
		}
		return path, true, nil
	}

	candidates := []string{}
	if strings.TrimSpace(opts.ExeDir) != "" {
		candidates = append(candidates, filepath.Join(opts.ExeDir, ConfigFileName))
	} else if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ConfigFileName))
	}
	programDataDir := strings.TrimSpace(opts.ProgramDataDir)
	if programDataDir == "" {
		programDataDir = `C:\ProgramData\kigrepair`
	}
	candidates = append(candidates, filepath.Join(programDataDir, ConfigFileName))
	if strings.TrimSpace(opts.WorkDir) != "" {
		candidates = append(candidates, filepath.Join(opts.WorkDir, ConfigFileName))
	} else if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, ConfigFileName))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			path, absErr := filepath.Abs(candidate)
			if absErr == nil {
				return path, false, nil
			}
			return candidate, false, nil
		}
	}
	return "", false, nil
}
