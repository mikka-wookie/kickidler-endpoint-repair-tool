package safety

import (
	"errors"
	"path/filepath"
	"strings"

	"kigrepair/internal/config"
)

var (
	ErrEmptyPath     = errors.New("path is empty")
	ErrRelativePath  = errors.New("path is relative")
	ErrUnresolvedEnv = errors.New("path contains unresolved environment variables")
	ErrDriveRoot     = errors.New("path points to a drive root")
	ErrUnknownPath   = errors.New("path is not in the allowed cleanup target list")
)

func ValidateCleanupPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrEmptyPath
	}

	expanded := filepath.Clean(config.ExpandPath(path))
	if config.HasUnresolvedEnv(expanded) {
		return ErrUnresolvedEnv
	}
	if !filepath.IsAbs(expanded) {
		return ErrRelativePath
	}
	if isDriveRoot(expanded) {
		return ErrDriveRoot
	}

	for _, allowed := range config.ExpandedCleanupPaths() {
		if strings.EqualFold(expanded, allowed) {
			return nil
		}
	}
	return ErrUnknownPath
}

func isDriveRoot(path string) bool {
	volume := filepath.VolumeName(path)
	if volume == "" {
		return false
	}
	rest := strings.TrimPrefix(path, volume)
	return rest == `\` || rest == `/` || rest == ""
}
