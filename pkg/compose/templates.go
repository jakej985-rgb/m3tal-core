package compose

import (
	"fmt"
	"strings"
)

// Template represents a Docker Compose template structure.
type Template struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
	Content     string            `json:"-"`
}

// Templates is the list of pre-defined Docker Compose templates.
var Templates = []Template{
	{
		Name:        "webapp",
		Description: "A standard web application stack with a backend app and a database",
		Parameters: map[string]string{
			"APP_NAME":    "Name of the application service (default: webapp)",
			"APP_IMAGE":   "Docker image for the application (default: nginx:latest)",
			"APP_PORT":    "External port exposed by the web app (default: 8080)",
			"DB_USER":     "Database user (default: admin)",
			"DB_PASSWORD": "Database password (default: secretpass)",
			"DB_NAME":     "Database name (default: appdb)",
		},
		Content: `version: '3.8'

services:
  ${APP_NAME:-webapp}:
    image: ${APP_IMAGE:-nginx:latest}
    restart: unless-stopped
    ports:
      - "${APP_PORT:-8080}:80"
    environment:
      - DATABASE_URL=postgres://${DB_USER:-admin}:${DB_PASSWORD:-secretpass}@db:5432/${DB_NAME:-appdb}
    depends_on:
      - db

  db:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_USER=${DB_USER:-admin}
      - POSTGRES_PASSWORD=${DB_PASSWORD:-secretpass}
      - POSTGRES_DB=${DB_NAME:-appdb}
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
`,
	},
	{
		Name:        "vpn-stack",
		Description: "A service stack routed through a VPN (Gluetun) container",
		Parameters: map[string]string{
			"VPN_USER":     "VPN provider username",
			"VPN_PASSWORD": "VPN provider password",
			"APP_NAME":     "Name of the application service (default: torrent)",
			"APP_IMAGE":    "Docker image for the application (default: lscr.io/linuxserver/transmission)",
			"APP_PORT":     "Web UI port inside the VPN (default: 9091)",
		},
		Content: `version: '3.8'

services:
  gluetun:
    image: qmcgaw/gluetun
    cap_add:
      - NET_ADMIN
    environment:
      - VPN_SERVICE_PROVIDER=custom
      - VPN_TYPE=openvpn
      - OPENVPN_USER=${VPN_USER}
      - OPENVPN_PASSWORD=${VPN_PASSWORD}
    ports:
      - "${APP_PORT:-9091}:${APP_PORT:-9091}"
    restart: unless-stopped

  ${APP_NAME:-torrent}:
    image: ${APP_IMAGE:-lscr.io/linuxserver/transmission}
    network_mode: "container:gluetun"
    restart: unless-stopped
    depends_on:
      - gluetun
`,
	},
}

// Generate interpolates parameters into a template and returns the valid compose YAML.
func Generate(templateName string, params map[string]string) ([]byte, error) {
	var tpl *Template
	for _, t := range Templates {
		if t.Name == templateName {
			tpl = &t
			break
		}
	}
	if tpl == nil {
		return nil, fmt.Errorf("template %q not found", templateName)
	}

	content := tpl.Content
	for k := range tpl.Parameters {
		val := params[k]
		content = resolvePlaceholder(content, k, val)
	}

	return []byte(content), nil
}

func resolvePlaceholder(content, param, val string) string {
	// 1. Resolve ${PARAM:-default}
	placeholderWithDefaultPrefix := fmt.Sprintf("${%s:-", param)
	for {
		idx := strings.Index(content, placeholderWithDefaultPrefix)
		if idx == -1 {
			break
		}
		endIdx := strings.Index(content[idx:], "}")
		if endIdx == -1 {
			break
		}
		actualEndIdx := idx + endIdx
		defaultValue := content[idx+len(placeholderWithDefaultPrefix) : actualEndIdx]

		replacement := val
		if replacement == "" {
			replacement = defaultValue
		}

		content = content[:idx] + replacement + content[actualEndIdx+1:]
	}

	// 2. Resolve ${PARAM}
	placeholder := fmt.Sprintf("${%s}", param)
	content = strings.ReplaceAll(content, placeholder, val)

	return content
}
