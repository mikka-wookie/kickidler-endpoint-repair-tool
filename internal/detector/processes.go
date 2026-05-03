package detector

type ProcessState struct {
	Name    string `json:"name"`
	Exists  bool   `json:"exists"`
	PID     int    `json:"pid,omitempty"`
	Message string `json:"message,omitempty"`
}
