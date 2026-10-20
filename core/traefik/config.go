package traefik

// ServiceConfig represents a service descriptor for generating routing labels.
type ServiceConfig struct {
	Name string
	Port int
}

// GenerateLabels produces the Traefik labels required for routing.
func GenerateLabels(s ServiceConfig) map[string]string {
	return map[string]string{
		"traefik.enable": "true",
		"traefik.http.routers." + s.Name + ".rule": "Host(`" + s.Name + ".local`)",
	}
}
