package installer

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
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
	validation := ValidateMSI(abs, ValidationOptions{
		ExplicitPath:        true,
		AllowUnknownName:    true,
		DeepMetadata:        false,
		SignatureCheck:      false,
		ExpectedProductCode: "",
		ExpectedPackedCode:  "",
	})
	if !validation.IsUsable() {
		return "", fmt.Errorf("%w: %s", ErrInvalidInstaller, validation.ErrorSummary())
	}
	return validation.Path, nil
}
