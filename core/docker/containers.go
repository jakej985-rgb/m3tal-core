package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// PortInfo represents container port mapping info
type PortInfo struct {
	IP          string `json:"ip,omitempty"`
	PrivatePort int    `json:"private_port"`
	PublicPort  int    `json:"public_port,omitempty"`
	Type        string `json:"type"`
}

// ContainerInfo represents basic container information
type ContainerInfo struct {
	ID       string            `json:"id"`
	Names    []string          `json:"names"`
	Image    string            `json:"image"`
	Status   string            `json:"status"`
	State    string            `json:"state"`
	CPU      float64           `json:"cpu"`
	Memory   float64           `json:"memory"`
	Labels   map[string]string `json:"labels,omitempty"`
	Ports    []PortInfo        `json:"ports,omitempty"`
	Networks []string          `json:"networks,omitempty"`
}

// ContainerEvent represents a container lifecycle event
type ContainerEvent struct {
	Action        string // "start", "stop", "die", etc.
	ContainerName string
}

// Provider defines the interface for container management
type Provider interface {
	ListContainers() ([]ContainerInfo, error)
	StartContainer(name string) error
	StopContainer(name string) error
	RestartContainer(name string) error
	Logs(name string, tail string) (string, error)
	SubscribeEvents(ctx context.Context) (<-chan ContainerEvent, error)
	StreamLogs(ctx context.Context, name string, tail string) (io.ReadCloser, error)

	ListImages() ([]models.DockerImage, error)
	PruneImages() (int64, error)
	ListVolumes() ([]models.DockerVolume, error)
	PruneVolumes() (int64, error)
	ListNetworks() ([]models.DockerNetwork, error)
	PruneNetworks() (int, error)
}

var globalProvider Provider

// SetProvider overrides the global container provider (Docker, Podman, Mock)
func SetProvider(p Provider) {
	globalProvider = p
}

// GetProvider returns the active container provider
func GetProvider() (Provider, error) {
	if globalProvider == nil {
		p, err := NewDockerManager()
		if err != nil {
			return nil, err
		}
		globalProvider = p
	}
	return globalProvider, nil
}

// DockerManager handles Docker container operations (implements Provider)
type DockerManager struct {
	cli *client.Client
}

// NewDockerManager creates a new Docker manager
func NewDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerManager{cli: cli}, nil
}

func (m *DockerManager) ListContainers() ([]ContainerInfo, error) {
	ctx := context.Background()
	containers, err := m.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var info []ContainerInfo
	for _, c := range containers {
		var ports []PortInfo
		for _, p := range c.Ports {
			ports = append(ports, PortInfo{
				IP:          p.IP,
				PrivatePort: int(p.PrivatePort),
				PublicPort:  int(p.PublicPort),
				Type:        p.Type,
			})
		}
		var networks []string
		if c.NetworkSettings != nil {
			for netName := range c.NetworkSettings.Networks {
				networks = append(networks, netName)
			}
		}
		info = append(info, ContainerInfo{
			ID:       c.ID,
			Names:    c.Names,
			Image:    c.Image,
			Status:   c.Status,
			State:    c.State,
			Labels:   c.Labels,
			Ports:    ports,
			Networks: networks,
		})
	}
	return info, nil
}

func (m *DockerManager) StartContainer(name string) error {
	ctx := context.Background()
	return m.cli.ContainerStart(ctx, name, types.ContainerStartOptions{})
}

func (m *DockerManager) StopContainer(name string) error {
	ctx := context.Background()
	return m.cli.ContainerStop(ctx, name, container.StopOptions{})
}

func (m *DockerManager) RestartContainer(name string) error {
	ctx := context.Background()
	return m.cli.ContainerRestart(ctx, name, container.StopOptions{})
}

func (m *DockerManager) Logs(name string, tail string) (string, error) {
	ctx := context.Background()
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	}
	result, err := m.cli.ContainerLogs(ctx, name, options)
	if err != nil {
		return "", err
	}
	defer result.Close()

	content, err := io.ReadAll(result)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (m *DockerManager) SubscribeEvents(ctx context.Context) (<-chan ContainerEvent, error) {
	msgCh, errCh := m.cli.Events(ctx, types.EventsOptions{})
	ch := make(chan ContainerEvent, 100)

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err != nil {
					return
				}
			case msg := <-msgCh:
				if string(msg.Type) == "container" {
					name := msg.Actor.Attributes["name"]
					if name == "" {
						name = msg.Actor.ID
					}
					ch <- ContainerEvent{
						Action:        string(msg.Action),
						ContainerName: name,
					}
				}
			}
		}
	}()

	return ch, nil
}

func (m *DockerManager) StreamLogs(ctx context.Context, name string, tail string) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       tail,
	}
	return m.cli.ContainerLogs(ctx, name, options)
}

func (m *DockerManager) ListImages() ([]models.DockerImage, error) {
	cmd := exec.Command("docker", "image", "ls", "--format", "{{json .}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var list []models.DockerImage
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var img struct {
			ID         string `json:"ID"`
			Repository string `json:"Repository"`
			Tag        string `json:"Tag"`
			Size       string `json:"Size"`
			CreatedAt  string `json:"CreatedAt"`
		}
		if err := json.Unmarshal([]byte(line), &img); err != nil {
			continue
		}

		var sizeBytes int64
		sizeStr := strings.ToUpper(img.Size)
		if strings.HasSuffix(sizeStr, "GB") {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStr, "GB"), 64)
			sizeBytes = int64(val * 1024 * 1024 * 1024)
		} else if strings.HasSuffix(sizeStr, "MB") {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStr, "MB"), 64)
			sizeBytes = int64(val * 1024 * 1024)
		} else if strings.HasSuffix(sizeStr, "KB") {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStr, "KB"), 64)
			sizeBytes = int64(val * 1024)
		} else if strings.HasSuffix(sizeStr, "B") {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStr, "B"), 64)
			sizeBytes = int64(val)
		}

		var createdUnix int64
		layout := "2006-01-02 15:04:05 -0700 MST"
		if t, err := time.Parse(layout, img.CreatedAt); err == nil {
			createdUnix = t.Unix()
		}

		list = append(list, models.DockerImage{
			ID:         img.ID,
			Repository: img.Repository,
			Tag:        img.Tag,
			Size:       sizeBytes,
			Created:    createdUnix,
		})
	}
	return list, nil
}

func (m *DockerManager) PruneImages() (int64, error) {
	out, err := exec.Command("docker", "image", "prune", "-a", "-f").Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Total reclaimed space:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				sizeStr := strings.TrimSpace(parts[1])
				sizeStrUpper := strings.ToUpper(sizeStr)
				var sizeBytes int64
				if strings.HasSuffix(sizeStrUpper, "GB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStrUpper, "GB"), 64)
					sizeBytes = int64(val * 1024 * 1024 * 1024)
				} else if strings.HasSuffix(sizeStrUpper, "MB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStrUpper, "MB"), 64)
					sizeBytes = int64(val * 1024 * 1024)
				} else if strings.HasSuffix(sizeStrUpper, "KB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStrUpper, "KB"), 64)
					sizeBytes = int64(val * 1024)
				}
				return sizeBytes, nil
			}
		}
	}
	return 0, nil
}

func (m *DockerManager) ListVolumes() ([]models.DockerVolume, error) {
	cmd := exec.Command("docker", "volume", "ls", "--format", "{{json .}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var list []models.DockerVolume
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var vol struct {
			Name       string `json:"Name"`
			Driver     string `json:"Driver"`
			Mountpoint string `json:"Mountpoint"`
			Labels     string `json:"Labels"`
		}
		if err := json.Unmarshal([]byte(line), &vol); err != nil {
			continue
		}

		labelsMap := make(map[string]string)
		if vol.Labels != "" {
			for _, labelPair := range strings.Split(vol.Labels, ",") {
				parts := strings.SplitN(labelPair, "=", 2)
				if len(parts) == 2 {
					labelsMap[parts[0]] = parts[1]
				} else if len(parts) == 1 {
					labelsMap[parts[0]] = ""
				}
			}
		}

		list = append(list, models.DockerVolume{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Labels:     labelsMap,
		})
	}
	return list, nil
}

func (m *DockerManager) PruneVolumes() (int64, error) {
	out, err := exec.Command("docker", "volume", "prune", "-f").Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Total reclaimed space:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				sizeStr := strings.TrimSpace(parts[1])
				sizeStrUpper := strings.ToUpper(sizeStr)
				var sizeBytes int64
				if strings.HasSuffix(sizeStrUpper, "GB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStrUpper, "GB"), 64)
					sizeBytes = int64(val * 1024 * 1024 * 1024)
				} else if strings.HasSuffix(sizeStrUpper, "MB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStrUpper, "MB"), 64)
					sizeBytes = int64(val * 1024 * 1024)
				} else if strings.HasSuffix(sizeStrUpper, "KB") {
					val, _ := strconv.ParseFloat(strings.TrimSuffix(sizeStrUpper, "KB"), 64)
					sizeBytes = int64(val * 1024)
				}
				return sizeBytes, nil
			}
		}
	}
	return 0, nil
}

func (m *DockerManager) ListNetworks() ([]models.DockerNetwork, error) {
	cmd := exec.Command("docker", "network", "ls", "--format", "{{json .}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var list []models.DockerNetwork
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var net struct {
			ID     string `json:"ID"`
			Name   string `json:"Name"`
			Driver string `json:"Driver"`
			Scope  string `json:"Scope"`
		}
		if err := json.Unmarshal([]byte(line), &net); err != nil {
			continue
		}

		list = append(list, models.DockerNetwork{
			ID:     net.ID,
			Name:   net.Name,
			Driver: net.Driver,
			Scope:  net.Scope,
		})
	}
	return list, nil
}

func (m *DockerManager) PruneNetworks() (int, error) {
	out, err := exec.Command("docker", "network", "prune", "-f").Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(out), "\n")
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(trimmed, "Deleted Networks:") {
			count++
		}
	}
	return count, nil
}

// MockProvider for testing (implements Provider)
type MockProvider struct {
	Containers []ContainerInfo
}

func (m *MockProvider) ListContainers() ([]ContainerInfo, error) { return m.Containers, nil }
func (m *MockProvider) StartContainer(name string) error         { return nil }
func (m *MockProvider) StopContainer(name string) error          { return nil }
func (m *MockProvider) RestartContainer(name string) error       { return nil }
func (m *MockProvider) Logs(name string, tail string) (string, error) {
	return "mock logs for " + name, nil
}
func (m *MockProvider) SubscribeEvents(ctx context.Context) (<-chan ContainerEvent, error) {
	ch := make(chan ContainerEvent)
	return ch, nil
}
func (m *MockProvider) StreamLogs(ctx context.Context, name string, tail string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("mock streaming logs for " + name + "\n")), nil
}

func (m *MockProvider) ListImages() ([]models.DockerImage, error)     { return nil, nil }
func (m *MockProvider) PruneImages() (int64, error)                   { return 0, nil }
func (m *MockProvider) ListVolumes() ([]models.DockerVolume, error)   { return nil, nil }
func (m *MockProvider) PruneVolumes() (int64, error)                  { return 0, nil }
func (m *MockProvider) ListNetworks() ([]models.DockerNetwork, error) { return nil, nil }
func (m *MockProvider) PruneNetworks() (int, error)                   { return 0, nil }
