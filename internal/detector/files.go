package detector

type FileState struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Message string `json:"message,omitempty"`
}
