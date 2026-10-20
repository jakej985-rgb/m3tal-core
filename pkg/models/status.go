package models

// Status represents the health status of system components.
type Status struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
	Details    map[string]string `json:"details,omitempty"`
}

// CheckResult represents the result of a single preflight diagnostic check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "PASS", "WARN", "FAIL"
	Message string `json:"message"`
}
