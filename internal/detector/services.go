package detector

type ServiceState struct {
	Name    string `json:"name"`
	Exists  bool   `json:"exists"`
	Running bool   `json:"running"`
	Message string `json:"message,omitempty"`
}
