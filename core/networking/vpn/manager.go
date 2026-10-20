package vpn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/jakej985-rgb/m3tal-core/core/docker"
	"github.com/jakej985-rgb/m3tal-core/pkg/system"
)

// Manager handles VPN container operations and status reporting.
type Manager struct {
	cli *client.Client
}

// NewManager creates a new VPN manager.
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Manager{cli: cli}, nil
}

// Start starts the Gluetun container.
func (m *Manager) Start() error {
	if m.cli == nil {
		return fmt.Errorf("docker client not initialized")
	}
	ctx := context.Background()
	return m.cli.ContainerStart(ctx, "gluetun", types.ContainerStartOptions{})
}

// Stop stops the Gluetun container.
func (m *Manager) Stop() error {
	if m.cli == nil {
		return fmt.Errorf("docker client not initialized")
	}
	ctx := context.Background()
	return m.cli.ContainerStop(ctx, "gluetun", container.StopOptions{})
}

// Restart restarts the Gluetun container.
func (m *Manager) Restart() error {
	if m.cli == nil {
		return fmt.Errorf("docker client not initialized")
	}
	ctx := context.Background()
	return m.cli.ContainerRestart(ctx, "gluetun", container.StopOptions{})
}

// GetStatus checks Gluetun health and retrieves its public IP and settings.
func (m *Manager) GetStatus() (*VPNStatus, error) {
	if m.cli == nil {
		return &VPNStatus{
			Connected:  false,
			StatusText: "Docker client not initialized",
		}, nil
	}
	ctx := context.Background()

	inspect, err := m.cli.ContainerInspect(ctx, "gluetun")
	if err != nil {
		return &VPNStatus{
			Connected:  false,
			StatusText: "Container 'gluetun' not found",
		}, nil
	}

	status := &VPNStatus{
		Connected:  inspect.State.Running,
		StatusText: inspect.State.Status,
	}

	// Extract provider and region from env vars
	for _, env := range inspect.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "VPN_SERVICE_PROVIDER":
			status.Provider = parts[1]
		case "SERVER_REGIONS":
			status.Region = parts[1]
		}
	}

	if !inspect.State.Running {
		return status, nil
	}

	// Try to get external IP from the container
	ip, err := m.GetOutboundIP()
	if err == nil {
		status.ExternalIP = ip
		status.Connected = true
	} else {
		status.Connected = false
		status.StatusText = fmt.Sprintf("Running but disconnected: %v", err)
	}

	// Check for forwarded port if PIA is used
	if strings.Contains(strings.ToLower(status.Provider), "private internet access") {
		if port, err := m.GetForwardedPort(); err == nil {
			status.ForwardedPort = port
		}
	}

	return status, nil
}

// GetOutboundIP queries the public IP through the Gluetun container network namespace.
func (m *Manager) GetOutboundIP() (string, error) {
	if m.cli == nil {
		return "", fmt.Errorf("docker client not initialized")
	}
	// Fallback 1: Control server
	clientHTTP := &http.Client{Timeout: 2 * time.Second}
	resp, err := clientHTTP.Get("http://localhost:8000/v1/publicip/ip")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			data, err := io.ReadAll(resp.Body)
			if err == nil {
				if strings.Contains(string(data), `"ip"`) {
					var result struct {
						IP string `json:"ip"`
					}
					if errJson := json.Unmarshal(data, &result); errJson == nil && result.IP != "" {
						return result.IP, nil
					}
				}
				cleanedIP := strings.TrimSpace(string(data))
				if !strings.Contains(cleanedIP, "{") {
					return cleanedIP, nil
				}
			}
		}
	}

	// Fallback 2: Execute command directly in Gluetun container
	ip, err := m.execInContainer("gluetun", []string{"wget", "-qO-", "http://ifconfig.me"})
	if err == nil && ip != "" {
		return strings.TrimSpace(ip), nil
	}

	// Fallback 3: Query using qbittorrent if running
	ip, err = m.execInContainer("qbittorrent", []string{"curl", "-s", "http://ifconfig.me"})
	if err == nil && ip != "" {
		return strings.TrimSpace(ip), nil
	}

	return "", fmt.Errorf("unable to query public IP through Gluetun network namespace")
}

// GetForwardedPort retrieves the forwarded port from the container.
func (m *Manager) GetForwardedPort() (int, error) {
	if m.cli == nil {
		return 0, fmt.Errorf("docker client not initialized")
	}
	// Port is typically written to /tmp/gluetun/forwarded_port
	out, err := m.execInContainer("gluetun", []string{"cat", "/tmp/gluetun/forwarded_port"})
	if err != nil {
		// Fallback path
		out, err = m.execInContainer("gluetun", []string{"cat", "/gluetun/forwarded_port"})
		if err != nil {
			return 0, err
		}
	}

	var port int
	_, err = fmt.Sscanf(strings.TrimSpace(out), "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("invalid port format in file: %w", err)
	}

	return port, nil
}

// SwitchRegion updates region in configuration and restarts stack.
func (m *Manager) SwitchRegion(region string) error {
	configPath := system.GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read env config: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "VPN_REGIONS=") {
			lines[i] = fmt.Sprintf("VPN_REGIONS=%s", region)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("VPN_REGIONS=%s", region))
	}

	err = os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to save env config: %w", err)
	}

	// Redeploy network stack to apply
	stackDir := system.GetStackDir()
	networkCompose := filepath.Join(stackDir, "network-compose.yml")
	if _, err := os.Stat(networkCompose); err == nil {
		_, errDeploy := docker.DeployStack(networkCompose, 0)
		return errDeploy
	}

	if m.cli == nil {
		return nil
	}

	return m.Restart()
}

// CheckLeak checks for VPN configuration leaks.
func (m *Manager) CheckLeak() (bool, string, string, error) {
	// Get host IP
	clientHTTP := &http.Client{Timeout: 3 * time.Second}
	resp, err := clientHTTP.Get("http://ifconfig.me")
	if err != nil {
		return false, "", "", fmt.Errorf("unable to fetch host public IP: %w", err)
	}
	defer resp.Body.Close()
	hostIPBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", "", err
	}
	hostIP := strings.TrimSpace(string(hostIPBytes))

	// Get VPN IP
	vpnIP, err := m.GetOutboundIP()
	if err != nil {
		return false, hostIP, "", fmt.Errorf("unable to fetch VPN outbound IP: %w", err)
	}

	isLeak := hostIP == vpnIP
	return isLeak, hostIP, vpnIP, nil
}

// StopDependentContainers finds and stops all containers connected to the Gluetun network namespace.
func (m *Manager) StopDependentContainers() ([]string, error) {
	if m.cli == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}
	ctx := context.Background()
	containersList, err := m.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var stopped []string
	for _, c := range containersList {
		inspect, err := m.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}

		// Check if it shares network namespace with Gluetun
		if inspect.HostConfig.NetworkMode == "container:gluetun" && inspect.State.Running {
			errStop := m.cli.ContainerStop(ctx, c.ID, container.StopOptions{})
			if errStop == nil {
				name := c.Names[0]
				name = strings.TrimPrefix(name, "/")
				stopped = append(stopped, name)
			}
		}
	}

	return stopped, nil
}

func (m *Manager) execInContainer(containerName string, cmd []string) (string, error) {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}

	execIDObj, err := m.cli.ContainerExecCreate(ctx, containerName, config)
	if err != nil {
		return "", err
	}

	resp, err := m.cli.ContainerExecAttach(ctx, execIDObj.ID, types.ExecStartCheck{})
	if err != nil {
		return "", err
	}
	defer resp.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}
