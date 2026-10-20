package models

// RouteRecord is a stored route mapping.
type RouteRecord struct {
	ID          int64  `json:"id"`
	Service     string `json:"service"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
	Entrypoints string `json:"entrypoints"`
	Stack       string `json:"stack,omitempty"`
	SSL         bool   `json:"ssl"`
	Middlewares string `json:"middlewares,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// DiscoverableService represents container proxy candidates.
type DiscoverableService struct {
	ContainerID   string   `json:"container_id"`
	ContainerName string   `json:"container_name"`
	Image         string   `json:"image"`
	State         string   `json:"state"`
	Ports         []int    `json:"ports"`
	Networks      []string `json:"networks"`
	Exposed       bool     `json:"exposed"`
	Domain        string   `json:"domain,omitempty"`
	SSL           bool     `json:"ssl,omitempty"`
	Middlewares   []string `json:"middlewares,omitempty"`
}
