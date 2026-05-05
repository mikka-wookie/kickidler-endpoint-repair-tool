package testfixtures

import (
	"strings"

	"kigrepair/internal/detector"
)

type FakeRegistryKey struct {
	Exists bool
	Error  string
}

func registryFromScenario(s Scenario) []detector.RegistryState {
	result := make([]detector.RegistryState, 0, len(s.RegistryKeys))
	for target, key := range s.RegistryKeys {
		root, path := splitRegistryTarget(target)
		result = append(result, detector.RegistryState{
			Root:   root,
			Path:   path,
			Exists: key.Exists,
			Error:  key.Error,
		})
	}
	return result
}

func splitRegistryTarget(target string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(target), `\`, 2)
	if len(parts) != 2 {
		return "", target
	}
	return parts[0], parts[1]
}
