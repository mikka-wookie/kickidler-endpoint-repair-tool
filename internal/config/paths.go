package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ExpandPath(path string) string {
	expanded := os.ExpandEnv(path)
	return percentEnvPattern.ReplaceAllStringFunc(expanded, func(match string) string {
		key := strings.Trim(match, "%")
		if value, ok := os.LookupEnv(key); ok && value != "" {
			return value
		}
		return match
	})
}

func ExpandedCleanupPaths() []string {
	paths := make([]string, 0, len(CleanupPaths))
	for _, path := range CleanupPaths {
		paths = append(paths, filepath.Clean(ExpandPath(path)))
	}
	return paths
}

func HasUnresolvedEnv(path string) bool {
	return strings.Contains(path, "%") || strings.Contains(path, "$")
}

var percentEnvPattern = regexp.MustCompile(`%[^%]+%`)
