package detector

import (
	"strings"

	svc "kigrepair/internal/services"
)

const wmiProviderService = "WmiProviderSE"

type ServiceState struct {
	Name                     string   `json:"name"`
	Exists                   bool     `json:"exists"`
	Status                   string   `json:"status,omitempty"`
	StartType                string   `json:"start_type,omitempty"`
	ImagePath                string   `json:"image_path,omitempty"`
	RawImagePath             string   `json:"raw_image_path,omitempty"`
	ExecutablePath           string   `json:"executable_path,omitempty"`
	NormalizedExecutablePath string   `json:"normalized_executable_path,omitempty"`
	Arguments                []string `json:"arguments,omitempty"`
	InstallRoot              string   `json:"install_root,omitempty"`
	InstallMode              string   `json:"install_mode,omitempty"`
	GrabberRelated           bool     `json:"grabber_related"`
	TrustLevel               string   `json:"trust_level"`
	Warnings                 []string `json:"warnings,omitempty"`
	Errors                   []string `json:"errors,omitempty"`
	ExpectedImagePathMatch   bool     `json:"expected_image_path_match"`
	Error                    string   `json:"error,omitempty"`
}

func DetectServices(system SystemState) []ServiceState {
	services := make([]ServiceState, 0, len(knownServiceNames))
	for _, name := range knownServiceNames {
		services = append(services, serviceStateFromInfo(svc.QueryService(name, nil), system))
	}
	return services
}

func HardenServiceStates(system SystemState, services []ServiceState) []ServiceState {
	result := make([]ServiceState, 0, len(services))
	for _, service := range services {
		if !service.Exists {
			if service.TrustLevel == "" {
				service.TrustLevel = svc.TrustUnknown
			}
			result = append(result, service)
			continue
		}
		raw := service.RawImagePath
		if raw == "" {
			raw = service.ImagePath
		}
		info := svc.ClassifyService(service.Name, service.Exists, service.Status, service.StartType, raw, service.Error)
		hardened := serviceStateFromInfo(info, system)
		if service.ImagePath != "" && hardened.ImagePath == "" {
			hardened.ImagePath = service.ImagePath
		}
		result = append(result, hardened)
	}
	return result
}

func serviceStateFromInfo(info svc.ServiceInfo, system SystemState) ServiceState {
	if info.Exists && info.ExecutablePath != "" {
		root, binaryDir, mode := classifyExecutablePath(system, info.ExecutablePath)
		if root != "" && mode != InstallModeUnknown {
			info.GrabberRelated = true
			info.TrustLevel = svc.TrustTrusted
			info.InstallRoot = root
			info.InstallMode = string(mode)
			if binaryDir != "" {
				info.NormalizedExecutablePath = svc.NormalizePath(info.ExecutablePath)
			}
		} else if svc.SupportedServiceName(info.Name) && info.RawImagePath != "" && info.TrustLevel != svc.TrustSupportedNameOnly && info.TrustLevel != svc.TrustQueryFailed {
			info.TrustLevel = svc.TrustPathMismatch
			info.GrabberRelated = false
		}
	}
	state := ServiceState{
		Name:                     info.Name,
		Exists:                   info.Exists,
		Status:                   info.State,
		StartType:                info.StartType,
		ImagePath:                info.RawImagePath,
		RawImagePath:             info.RawImagePath,
		ExecutablePath:           info.ExecutablePath,
		NormalizedExecutablePath: info.NormalizedExecutablePath,
		Arguments:                append([]string{}, info.Arguments...),
		InstallRoot:              info.InstallRoot,
		InstallMode:              info.InstallMode,
		GrabberRelated:           info.GrabberRelated,
		TrustLevel:               info.TrustLevel,
		Warnings:                 append([]string{}, info.Warnings...),
		Errors:                   append([]string{}, info.Errors...),
	}
	if len(state.Errors) > 0 {
		state.Error = strings.Join(state.Errors, "; ")
	}
	if strings.EqualFold(info.Name, wmiProviderService) {
		state.ExpectedImagePathMatch = pathsEqual(state.ExecutablePath, system.SystemRoot+`\System32\wmi\bin\svchost.exe`) ||
			pathsEqual(state.ExecutablePath, system.SystemRoot+`\System32\wmi\bin\WmiPrvSE.exe`) ||
			pathsEqual(state.ExecutablePath, system.SystemRoot+`\System32\wmi\bin\RuntimeBroker.exe`)
	}
	return state
}

func pathsEqual(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
}

func normalizePath(path string) string {
	return strings.ToLower(svc.NormalizePath(path))
}

func NormalizeWindowsPath(path string) string {
	return svc.NormalizePath(path)
}
