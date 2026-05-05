package testfixtures

import (
	"kigrepair/internal/detector"
)

type FakeProcess struct {
	PID            int
	Name           string
	ExecutablePath string
	CommandLine    string
}

func processesFromScenario(s Scenario, system detector.SystemState) []detector.ProcessState {
	result := make([]detector.ProcessState, 0, len(s.Processes))
	for _, process := range s.Processes {
		classified := detector.ClassifyProcess(detector.RawProcessInfo{
			ProcessID:      process.PID,
			Name:           process.Name,
			ExecutablePath: process.ExecutablePath,
			CommandLine:    process.CommandLine,
		}, detector.MatchOptions{System: system})
		if classified.GrabberRelated || len(classified.Warnings) > 0 {
			result = append(result, classified)
		}
	}
	return result
}
