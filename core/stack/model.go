package stack

// Stack represents a multi-container deployment stack configuration.
type Stack struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ComposePath string    `json:"compose_path"`
	Status      string    `json:"status"`
	Services    []Service `json:"services,omitempty"`
}

// Service represents an individual service within a stack.
type Service struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}
