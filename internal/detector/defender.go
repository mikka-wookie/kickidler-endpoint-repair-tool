package detector

type DefenderState struct {
	Available  bool     `json:"available"`
	Enabled    bool     `json:"enabled"`
	Exclusions []string `json:"exclusions,omitempty"`
	Message    string   `json:"message,omitempty"`
}
