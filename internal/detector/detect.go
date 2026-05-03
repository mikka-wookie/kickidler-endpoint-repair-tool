package detector

import "time"

type DetectionReport struct {
	Health    GrabberHealthStatus `json:"health"`
	Services  []ServiceState      `json:"services"`
	Processes []ProcessState      `json:"processes"`
	Files     []FileState         `json:"files"`
	Registry  []RegistryState     `json:"registry"`
	Defender  DefenderState       `json:"defender"`
	CreatedAt time.Time           `json:"created_at"`
}

func Detect() DetectionReport {
	return DetectionReport{
		Health:    GrabberHealthUnknown,
		Defender:  DefenderState{Available: false, Message: "placeholder"},
		CreatedAt: time.Now(),
	}
}
