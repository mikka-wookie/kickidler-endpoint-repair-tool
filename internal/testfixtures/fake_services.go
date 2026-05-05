package testfixtures

import (
	"kigrepair/internal/detector"
)

type FakeService struct {
	Name      string
	Exists    bool
	Status    string
	StartType string
	ImagePath string
	Error     string
}

func servicesFromScenario(s Scenario) []detector.ServiceState {
	result := make([]detector.ServiceState, 0, len(s.Services))
	for _, service := range s.Services {
		imagePath := service.ImagePath
		result = append(result, detector.ServiceState{
			Name:         service.Name,
			Exists:       service.Exists,
			Status:       service.Status,
			StartType:    service.StartType,
			ImagePath:    imagePath,
			RawImagePath: imagePath,
			Error:        service.Error,
		})
	}
	return result
}
