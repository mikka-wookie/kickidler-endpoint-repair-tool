package detector

type RegistryState struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Message string `json:"message,omitempty"`
}
