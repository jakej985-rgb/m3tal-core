package docker

import (
	"testing"

	"github.com/jakej985-rgb/m3tal-core/core/traefik"
)

var testCompose = `
services:
  app:
    image: nginx:latest
    ports:
      - "8080:80"
      - "127.0.0.1:9090:443"
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.app.rule=Host(` + "`" + `app.local` + "`" + `)"
      - "traefik.http.routers.app.entrypoints=web"
      - "traefik.http.services.app.loadbalancer.server.port=80"
      - "m3tal.stack=media"
    environment:
      - PUID=1000
      - PGID=1000
    networks:
      - proxy
    restart: unless-stopped

  redis:
    image: redis:7
    ports:
      - "6379:6379"

networks:
  proxy:
    external: true
`

func TestParseComposeBytes(t *testing.T) {
	cf, err := traefik.ParseComposeBytes([]byte(testCompose))
	if err != nil {
		t.Fatalf("ParseComposeBytes failed: %v", err)
	}

	if len(cf.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cf.Services))
	}

	names := cf.ServiceNames()
	if len(names) != 2 || names[0] != "app" || names[1] != "redis" {
		t.Errorf("unexpected service names: %v", names)
	}
}

func TestGetTraefikLabels(t *testing.T) {
	cf, _ := traefik.ParseComposeBytes([]byte(testCompose))
	labels := cf.GetTraefikLabels("app")

	if labels["traefik.enable"] != "true" {
		t.Errorf("expected traefik.enable=true, got %q", labels["traefik.enable"])
	}
	if len(labels) != 4 {
		t.Errorf("expected 4 traefik labels, got %d: %v", len(labels), labels)
	}

	// redis should have no traefik labels
	redisLabels := cf.GetTraefikLabels("redis")
	if len(redisLabels) != 0 {
		t.Errorf("expected 0 traefik labels for redis, got %d", len(redisLabels))
	}

	// nonexistent service should return nil
	if cf.GetTraefikLabels("nonexistent") != nil {
		t.Error("expected nil for nonexistent service")
	}
}

func TestGetExposedPorts(t *testing.T) {
	cf, _ := traefik.ParseComposeBytes([]byte(testCompose))

	appPorts := cf.GetExposedPorts("app")
	if len(appPorts) != 2 {
		t.Fatalf("expected 2 ports for app, got %d: %v", len(appPorts), appPorts)
	}
	if appPorts[0] != 8080 {
		t.Errorf("expected first port 8080, got %d", appPorts[0])
	}
	if appPorts[1] != 9090 {
		t.Errorf("expected second port 9090, got %d", appPorts[1])
	}

	redisPorts := cf.GetExposedPorts("redis")
	if len(redisPorts) != 1 || redisPorts[0] != 6379 {
		t.Errorf("expected redis port 6379, got %v", redisPorts)
	}
}

func TestGetContainerPort(t *testing.T) {
	cf, _ := traefik.ParseComposeBytes([]byte(testCompose))

	if p := cf.GetContainerPort("app"); p != 80 {
		t.Errorf("expected container port 80, got %d", p)
	}
	if p := cf.GetContainerPort("redis"); p != 6379 {
		t.Errorf("expected container port 6379, got %d", p)
	}
	if p := cf.GetContainerPort("nonexistent"); p != 0 {
		t.Errorf("expected 0 for nonexistent, got %d", p)
	}
}

func TestGenerateLabels(t *testing.T) {
	labels := traefik.GenerateRouteLabels(traefik.RouteInput{
		Service: "app",
		Domain:  "app.local",
		Port:    8080,
	})

	if labels["traefik.enable"] != "true" {
		t.Error("missing traefik.enable")
	}
	if labels["traefik.http.routers.app.rule"] != "Host(`app.local`)" {
		t.Errorf("unexpected rule: %s", labels["traefik.http.routers.app.rule"])
	}
	if labels["traefik.http.routers.app.entrypoints"] != "web" {
		t.Error("missing entrypoints")
	}
	if labels["traefik.http.services.app.loadbalancer.server.port"] != "8080" {
		t.Error("missing port")
	}
	if labels["traefik.docker.network"] != "proxy" {
		t.Error("missing network")
	}
}

func TestGenerateMiddlewareLabels(t *testing.T) {
	labels := traefik.GenerateMiddlewareLabels(traefik.MiddlewareInput{
		Name: "auth",
		Type: "basicauth",
		Config: map[string]string{
			"users": "admin:$apr1$...",
		},
	})

	if labels["traefik.http.middlewares.auth.basicauth.users"] != "admin:$apr1$..." {
		t.Errorf("unexpected basicauth label: %v", labels)
	}
}

func TestValidateRoute(t *testing.T) {
	// Valid route
	errs := ValidateRoute(traefik.RouteInput{Service: "app", Domain: "app.local", Port: 8080})
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	// Missing service
	errs = ValidateRoute(traefik.RouteInput{Service: "", Domain: "app.local", Port: 8080})
	if len(errs) == 0 {
		t.Error("expected error for empty service")
	}

	// Invalid domain
	errs = ValidateRoute(traefik.RouteInput{Service: "app", Domain: "not a domain!", Port: 8080})
	if len(errs) == 0 {
		t.Error("expected error for invalid domain")
	}

	// Invalid port
	errs = ValidateRoute(traefik.RouteInput{Service: "app", Domain: "app.local", Port: 0})
	if len(errs) == 0 {
		t.Error("expected error for port 0")
	}

	// Reserved port
	errs = ValidateRoute(traefik.RouteInput{Service: "app", Domain: "app.local", Port: 22})
	if len(errs) == 0 {
		t.Error("expected error for reserved port 22")
	}
}

func TestValidateDomainUnique(t *testing.T) {
	existing := []traefik.RouteInput{
		{Service: "app", Domain: "app.local", Port: 8080},
	}

	errs := ValidateDomainUnique("app.local", existing)
	if len(errs) == 0 {
		t.Error("expected duplicate domain error")
	}

	errs = ValidateDomainUnique("other.local", existing)
	if len(errs) != 0 {
		t.Errorf("expected no errors for unique domain, got %v", errs)
	}
}

func TestValidateCompose(t *testing.T) {
	cf, _ := traefik.ParseComposeBytes([]byte(testCompose))
	errs := ValidateCompose(cf)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid compose, got %v", errs)
	}

	// Empty compose
	emptyCf := &traefik.ComposeFile{Services: map[string]traefik.Service{}}
	errs = ValidateCompose(emptyCf)
	if len(errs) == 0 {
		t.Error("expected error for empty compose")
	}
}

func TestMapLabels(t *testing.T) {
	mapCompose := `
services:
  web:
    image: nginx
    labels:
      traefik.enable: "true"
      traefik.http.routers.web.rule: "Host(` + "`" + `web.local` + "`" + `)"
`
	cf, err := traefik.ParseComposeBytes([]byte(mapCompose))
	if err != nil {
		t.Fatalf("failed to parse map-style labels: %v", err)
	}

	labels := cf.GetTraefikLabels("web")
	if labels["traefik.enable"] != "true" {
		t.Error("expected traefik.enable from map-style labels")
	}
}
