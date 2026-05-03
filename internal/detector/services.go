package detector

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const wmiProviderService = "WmiProviderSE"

type ServiceState struct {
	Name                   string `json:"name"`
	Exists                 bool   `json:"exists"`
	Status                 string `json:"status,omitempty"`
	StartType              string `json:"start_type,omitempty"`
	ImagePath              string `json:"image_path,omitempty"`
	ExpectedImagePathMatch bool   `json:"expected_image_path_match"`
	Error                  string `json:"error,omitempty"`
}

func DetectServices(system SystemState) []ServiceState {
	services := make([]ServiceState, 0, len(knownServiceNames))
	for _, name := range knownServiceNames {
		state := detectService(name)
		if strings.EqualFold(name, wmiProviderService) {
			state.ExpectedImagePathMatch = pathsEqual(state.ImagePath, expectedWMIServiceImagePath(system))
		}
		services = append(services, state)
	}
	return services
}

func detectService(name string) ServiceState {
	state := ServiceState{Name: name}
	query, err := exec.Command("sc.exe", "query", name).CombinedOutput()
	if err != nil {
		if bytes.Contains(query, []byte("1060")) || strings.Contains(strings.ToLower(string(query)), "does not exist") {
			state.Exists = false
			return state
		}
		state.Error = strings.TrimSpace(string(query))
		if state.Error == "" {
			state.Error = err.Error()
		}
		return state
	}
	state.Exists = true
	state.Status = parseSCStatus(string(query))

	qc, err := exec.Command("sc.exe", "qc", name).CombinedOutput()
	if err != nil {
		state.Error = strings.TrimSpace(string(qc))
		if state.Error == "" {
			state.Error = err.Error()
		}
		return state
	}
	state.StartType = parseSCValue(string(qc), "START_TYPE")
	state.ImagePath = parseSCValue(string(qc), "BINARY_PATH_NAME")
	return state
}

func parseSCStatus(output string) string {
	re := regexp.MustCompile(`(?m)^\s*STATE\s*:\s*\d+\s+([A-Z_]+)`)
	if match := re.FindStringSubmatch(output); len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return ""
}

func parseSCValue(output, key string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*:\s*(.*)$`)
	if match := re.FindStringSubmatch(output); len(match) == 2 {
		value := strings.TrimSpace(match[1])
		if key == "START_TYPE" {
			parts := strings.Fields(value)
			if len(parts) > 1 {
				return strings.ToLower(strings.Join(parts[1:], " "))
			}
		}
		return value
	}
	return ""
}

func expectedWMIServiceImagePath(system SystemState) string {
	return filepath.Join(system.SystemRoot, "System32", "wmi", "bin", "svchost.exe")
}

func pathsEqual(a, b string) bool {
	return normalizePath(a) == normalizePath(b)
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	path = expandWindowsEnv(path)
	path = strings.TrimPrefix(path, `\??\`)
	path = strings.ReplaceAll(path, "/", `\`)
	return strings.ToLower(filepath.Clean(path))
}

func expandWindowsEnv(value string) string {
	re := regexp.MustCompile(`%([^%]+)%`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		name := strings.Trim(match, "%")
		if expanded := os.Getenv(name); expanded != "" {
			return expanded
		}
		return match
	})
}
