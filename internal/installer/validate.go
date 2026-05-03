package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kigrepair/internal/detector"
)

var (
	ErrInvalidInvite    = errors.New("invalid invite")
	ErrInvalidInstaller = errors.New("invalid installer")
)

func ValidateInvite(value string) (string, error) {
	invite := strings.TrimSpace(value)
	if invite == "" {
		return "", fmt.Errorf("%w: invite is required", ErrInvalidInvite)
	}
	if strings.ContainsAny(invite, "\"'&|;><`") {
		return "", fmt.Errorf("%w: invite contains unsupported characters", ErrInvalidInvite)
	}
	return invite, nil
}

func ValidateInstallerPath(value string) (string, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		return "", fmt.Errorf("%w: installer path is required", ErrInvalidInstaller)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidInstaller, err)
	}
	abs = detector.NormalizeWindowsPath(abs)
	if !strings.EqualFold(filepath.Ext(abs), ".msi") {
		return "", fmt.Errorf("%w: installer must be an .msi file", ErrInvalidInstaller)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: installer file does not exist", ErrInvalidInstaller)
		}
		return "", fmt.Errorf("%w: %v", ErrInvalidInstaller, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: installer path is a directory", ErrInvalidInstaller)
	}
	return abs, nil
}
