package services

import (
	"path/filepath"
	"strings"

	"kigrepair/internal/config"
)

func SupportedServiceName(name string) bool {
	for _, supported := range config.Services {
		if strings.EqualFold(name, supported) {
			return true
		}
	}
	return false
}

func ClassifyService(name string, exists bool, state string, startType string, rawImagePath string, queryErr string) ServiceInfo {
	info := ServiceInfo{
		Name:           name,
		Exists:         exists,
		State:          NormalizeState(state),
		StartType:      strings.TrimSpace(startType),
		RawImagePath:   strings.TrimSpace(rawImagePath),
		GrabberRelated: false,
		TrustLevel:     TrustUnknown,
	}
	if queryErr != "" {
		info.Errors = append(info.Errors, strings.TrimSpace(queryErr))
		info.TrustLevel = TrustQueryFailed
		return info
	}
	if !exists {
		info.TrustLevel = TrustUnknown
		return info
	}
	if !SupportedServiceName(name) {
		info.TrustLevel = TrustUnknown
		info.Warnings = append(info.Warnings, "Service name is not in the supported Grabber allowlist")
		return info
	}
	if strings.TrimSpace(rawImagePath) == "" {
		info.GrabberRelated = true
		info.TrustLevel = TrustSupportedNameOnly
		info.Warnings = append(info.Warnings, "Supported service name exists but ImagePath is missing or unreadable")
		return info
	}

	parsed := ParseServiceImagePath(rawImagePath)
	info.ExecutablePath = parsed.ExecutablePath
	info.NormalizedExecutablePath = parsed.NormalizedPath
	info.Arguments = append([]string{}, parsed.Arguments...)
	info.Warnings = append(info.Warnings, parsed.Warnings...)
	if !parsed.Valid {
		info.GrabberRelated = true
		info.TrustLevel = TrustSupportedNameOnly
		if parsed.Error != "" {
			info.Errors = append(info.Errors, parsed.Error)
		}
		return info
	}

	root, mode, trusted := DeriveInstallRootAndMode(parsed.NormalizedPath)
	if trusted {
		info.GrabberRelated = true
		info.TrustLevel = TrustTrusted
		info.InstallRoot = root
		info.InstallMode = mode
		return info
	}

	info.TrustLevel = TrustPathMismatch
	info.Warnings = append(info.Warnings, "Supported service name exists but executable path is outside known Grabber paths. Service will not be modified automatically.")
	return info
}

func NormalizeState(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	state = strings.ReplaceAll(state, " ", "_")
	switch state {
	case StateRunning, StateStopped, StateStartPending, StateStopPending, StatePaused, StatePausePending, StateContinuePending:
		return state
	default:
		if state == "" {
			return ""
		}
		return StateUnknown
	}
}

func DeriveInstallRootAndMode(exePath string) (string, string, bool) {
	normalized := NormalizePath(exePath)
	for _, candidate := range standardRoots() {
		root := NormalizePath(candidate.root)
		if PathWithin(normalized, root) {
			return root, candidate.mode, true
		}
	}
	for _, allowed := range config.KnownWMIExecutablePaths {
		expanded := NormalizePath(config.ExpandPath(allowed))
		if SamePath(normalized, expanded) {
			return hiddenWMIRoot(), ModeHiddenWMI, true
		}
	}
	return "", ModeUnknown, false
}

func IsTrustedService(info ServiceInfo, allowNameOnly bool) bool {
	if !info.Exists || !SupportedServiceName(info.Name) {
		return false
	}
	if info.TrustLevel == TrustTrusted {
		return true
	}
	return allowNameOnly && info.TrustLevel == TrustSupportedNameOnly
}

func hiddenWMIRoot() string {
	return NormalizePath(config.ExpandPath(`%SystemRoot%\System32\wmi`))
}

type rootMode struct {
	root string
	mode string
}

func standardRoots() []rootMode {
	return []rootMode{
		{root: config.ExpandPath(`%ProgramFiles%\TeleLinkSoft`), mode: ModeStandard},
		{root: filepath.Join(config.ExpandPath(`%ProgramFiles%\TeleLinkSoft`), "bin"), mode: ModeStandard},
		{root: config.ExpandPath(`%ProgramFiles%\TeleLinkSoftHelper`), mode: ModeHelper},
		{root: config.ExpandPath(`%ProgramFiles(x86)%\TeleLinkSoft`), mode: ModeStandard},
		{root: filepath.Join(config.ExpandPath(`%ProgramFiles(x86)%\TeleLinkSoft`), "bin"), mode: ModeStandard},
		{root: config.ExpandPath(`%ProgramFiles(x86)%\TeleLinkSoftHelper`), mode: ModeHelper},
	}
}
