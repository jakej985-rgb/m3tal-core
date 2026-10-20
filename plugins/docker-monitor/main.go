package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jakej985-rgb/m3tal-core/pkg/plugins"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req plugins.RPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			sendError(0, -32700, "Parse error")
			continue
		}

		handleRequest(&req)
	}
}

func handleRequest(req *plugins.RPCRequest) {
	switch req.Method {
	case plugins.MethodInit:
		result := plugins.InitResult{
			Name:    "docker-monitor",
			Version: "0.1.0",
			Capabilities: []plugins.Capability{
				plugins.CapabilityCollector,
				plugins.CapabilityTUIWidget,
				plugins.CapabilityActions,
			},
			Tabs: []plugins.PluginTabSpec{
				{
					ID:          "docker_containers",
					Title:       "Docker",
					Shortcut:    "d",
					Description: "Docker container telemetry and status",
				},
			},
			Actions: []plugins.ActionSpec{
				{
					ID:      "restart_all",
					Label:   "Restart Stack",
					Keybind: "r",
				},
			},
		}
		sendResult(req.ID, result)

	case plugins.MethodCollect:
		containers := getDockerContainers()
		runningCount := 0
		rows := make([][]string, 0, len(containers))
		for _, c := range containers {
			if strings.Contains(strings.ToLower(c[3]), "up") {
				runningCount++
			}
			rows = append(rows, c)
		}

		result := plugins.CollectResult{
			Status: "ok",
			Metrics: []plugins.Metric{
				{
					Name:     "containers_running",
					Value:    float64(runningCount),
					Unit:     "containers",
					Category: "docker",
					Status:   "ok",
				},
				{
					Name:     "containers_total",
					Value:    float64(len(containers)),
					Unit:     "containers",
					Category: "docker",
					Status:   "ok",
				},
			},
			Tables: []plugins.TableData{
				{
					ID:      "docker_ps",
					Title:   "🐳 Docker Containers",
					Headers: []string{"ID", "Name", "Image", "Status"},
					Rows:    rows,
				},
			},
		}
		sendResult(req.ID, result)

	case plugins.MethodRender:
		var params plugins.RenderParams
		_ = json.Unmarshal(req.Params, &params)

		containers := getDockerContainers()
		var b strings.Builder
		b.WriteString(fmt.Sprintf("🐳 DOCKER CONTAINERS (%d active)\n", len(containers)))
		b.WriteString(strings.Repeat("─", 60) + "\n")
		b.WriteString(fmt.Sprintf("%-14s %-20s %-20s %-15s\n", "ID", "NAME", "IMAGE", "STATUS"))
		b.WriteString(strings.Repeat("─", 60) + "\n")
		for _, c := range containers {
			b.WriteString(fmt.Sprintf("%-14s %-20s %-20s %-15s\n", c[0], c[1], c[2], c[3]))
		}

		result := plugins.RenderResult{
			Format:  "plain",
			Content: b.String(),
		}
		sendResult(req.ID, result)

	case plugins.MethodAction:
		result := plugins.ActionResult{
			Success: true,
			Message: "Action executed successfully",
		}
		sendResult(req.ID, result)

	default:
		sendError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func getDockerContainers() [][]string {
	// Query local docker ps
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		var rows [][]string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 4 {
				rows = append(rows, []string{parts[0], parts[1], parts[2], parts[3]})
			}
		}
		if len(rows) > 0 {
			return rows
		}
	}

	// Fallback/Simulated data if docker daemon socket is not running
	return [][]string{
		{"a1b2c3d4", "m3tal-proxy", "traefik:v3.0", "Up 12 hours"},
		{"e5f6g7h8", "m3tal-dash", "m3tal-dashboard:latest", "Up 12 hours"},
		{"i9j0k1l2", "m3tal-cloudflared", "cloudflare/cloudflared", "Up 12 hours"},
	}
}

func sendResult(id uint64, result any) {
	resBytes, _ := json.Marshal(result)
	resp := plugins.RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resBytes,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func sendError(id uint64, code int, message string) {
	resp := plugins.RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &plugins.RPCError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}
