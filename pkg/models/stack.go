package models

// Stack represents compose stack information.
type Stack struct {
	Name        string   `json:"name"`
	ComposePath string   `json:"compose_path"`
	Services    []string `json:"services,omitempty"`
	Status      string   `json:"status"`
}
