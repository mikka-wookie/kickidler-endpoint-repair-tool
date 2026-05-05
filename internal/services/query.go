package services

import (
	"bytes"
	"errors"
	"os/exec"
	"regexp"
	"strings"
)

func QueryService(name string, runner CommandRunner) ServiceInfo {
	if runner == nil {
		runner = RunCommand
	}
	query := runner("sc.exe", "query", name)
	if query.ExitCode != 0 {
		output := strings.TrimSpace(query.Output)
		if serviceMissing(output) {
			return ClassifyService(name, false, "", "", "", "")
		}
		if output == "" && query.Err != nil {
			output = query.Err.Error()
		}
		return ClassifyService(name, false, "", "", "", output)
	}

	state := parseSCStatus(query.Output)
	qc := runner("sc.exe", "qc", name)
	if qc.ExitCode != 0 {
		output := strings.TrimSpace(qc.Output)
		if output == "" && qc.Err != nil {
			output = qc.Err.Error()
		}
		return ClassifyService(name, true, state, "", "", output)
	}
	startType := parseSCValue(qc.Output, "START_TYPE")
	imagePath := parseSCValue(qc.Output, "BINARY_PATH_NAME")
	return ClassifyService(name, true, state, startType, imagePath, "")
}

func QuerySupportedServices(runner CommandRunner) []ServiceInfo {
	result := make([]ServiceInfo, 0, len(supportedNames()))
	for _, name := range supportedNames() {
		result = append(result, QueryService(name, runner))
	}
	return result
}

func RunCommand(name string, args ...string) CommandResult {
	output, err := exec.Command(name, args...).CombinedOutput()
	code := 0
	if err != nil {
		code = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
	}
	return CommandResult{ExitCode: code, Output: strings.TrimSpace(string(output)), Err: err}
}

func parseSCStatus(output string) string {
	re := regexp.MustCompile(`(?m)^\s*STATE\s*:\s*\d+\s+([A-Z_]+)`)
	if match := re.FindStringSubmatch(output); len(match) == 2 {
		return NormalizeState(match[1])
	}
	return StateUnknown
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

func serviceMissing(output string) bool {
	lower := strings.ToLower(output)
	return bytes.Contains([]byte(output), []byte("1060")) || strings.Contains(lower, "does not exist")
}

func supportedNames() []string {
	return []string{"ngs", "tls", "WmiProviderSE"}
}
