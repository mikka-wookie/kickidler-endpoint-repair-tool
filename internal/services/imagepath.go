package services

import (
	"path/filepath"
	"regexp"
	"strings"

	"kigrepair/internal/config"
)

func ParseServiceImagePath(raw string) ParsedImagePath {
	result := ParsedImagePath{Raw: raw}
	value := strings.TrimSpace(raw)
	if value == "" {
		result.Error = "image path is empty"
		return result
	}

	var exe string
	var rest string
	if strings.HasPrefix(value, `"`) {
		end := strings.Index(value[1:], `"`)
		if end < 0 {
			result.Error = "image path has malformed quoting"
			result.Warnings = append(result.Warnings, "Opening quote has no closing quote")
			return result
		}
		exe = value[1 : 1+end]
		rest = strings.TrimSpace(value[1+end+1:])
	} else {
		lower := strings.ToLower(value)
		idx := strings.Index(lower, ".exe")
		if idx < 0 {
			fields := strings.Fields(value)
			if len(fields) == 0 {
				result.Error = "image path is empty"
				return result
			}
			exe = strings.Trim(fields[0], `"`)
			if len(fields) > 1 {
				rest = strings.Join(fields[1:], " ")
			}
		} else {
			exe = strings.TrimSpace(value[:idx+4])
			rest = strings.TrimSpace(value[idx+4:])
		}
	}

	normalized := NormalizePath(exe)
	if normalized == "" {
		result.Error = "executable path is empty"
		return result
	}
	result.ExecutablePath = normalized
	result.NormalizedPath = normalized
	result.Arguments = splitArguments(rest)
	result.Valid = true
	return result
}

func NormalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"'`)
	path = strings.TrimSpace(path)
	path = config.ExpandPath(path)
	path = strings.TrimPrefix(path, `\??\`)
	path = normalizeLongPathPrefix(path)
	path = strings.ReplaceAll(path, "/", `\`)
	path = collapseWindowsBackslashes(path)
	path = filepath.Clean(path)
	if path == "." {
		return ""
	}
	path = strings.TrimSuffix(path, `\`)
	if len(path) == 2 && path[1] == ':' {
		path += `\`
	}
	return path
}

func SamePath(a, b string) bool {
	return strings.EqualFold(NormalizePath(a), NormalizePath(b))
}

func PathWithin(child, parent string) bool {
	child = strings.ToLower(NormalizePath(child))
	parent = strings.ToLower(NormalizePath(parent))
	if child == "" || parent == "" {
		return false
	}
	if child == parent {
		return true
	}
	if strings.HasSuffix(parent, `\`) {
		return strings.HasPrefix(child, parent)
	}
	return strings.HasPrefix(child, parent+`\`)
}

func splitArguments(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}

func normalizeLongPathPrefix(path string) string {
	if strings.HasPrefix(path, `\\?\UNC\`) {
		return `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
	}
	if strings.HasPrefix(path, `\\?\`) {
		return strings.TrimPrefix(path, `\\?\`)
	}
	return path
}

func collapseWindowsBackslashes(path string) string {
	if strings.HasPrefix(path, `\\`) {
		return `\\` + collapseRepeatedBackslashes(path[2:])
	}
	return collapseRepeatedBackslashes(path)
}

func collapseRepeatedBackslashes(path string) string {
	re := regexp.MustCompile(`\\+`)
	return re.ReplaceAllString(path, `\`)
}
