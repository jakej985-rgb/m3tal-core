package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// Client handles authenticated HTTP communications with the M3TAL API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient instantiates a new Client.
func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:5050"
	}
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

// APIResponse represents the standardized JSON structure for M3TAL API responses.
type APIResponse struct {
	Status string                `json:"status"`
	Data   json.RawMessage       `json:"data"`
	Meta   json.RawMessage       `json:"meta,omitempty"`
	Error  *models.ErrorResponse `json:"error"`
}

func (c *Client) request(method, path string, body any, target any) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	timeout := 15 * time.Second
	if method == "POST" || strings.Contains(path, "/doctor") || strings.Contains(path, "/check-leak") {
		timeout = 15 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("X-API-Token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fmt.Errorf("API returned non-JSON response (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	if apiResp.Status == "error" || apiResp.Error != nil {
		if apiResp.Error != nil {
			return fmt.Errorf("API error (%s): %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return fmt.Errorf("API request returned error status")
	}

	if target != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, target); err != nil {
			return fmt.Errorf("failed to unmarshal API data field: %w", err)
		}
	}

	return nil
}

// Request executes a generic API request.
func (c *Client) Request(method, path string, body any, target any) error {
	return c.request(method, path, body, target)
}

// GetStats returns the host system metrics response.
func (c *Client) GetStats() (*models.MetricsResponse, error) {
	var metrics models.MetricsResponse
	err := c.request("GET", "/api/v2/system/stats", nil, &metrics)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// GetConfig returns the active sanitized daemon environment configuration.
func (c *Client) GetConfig() (map[string]string, error) {
	var config map[string]string
	err := c.request("GET", "/api/config", nil, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetCloudflaredConfig fetches the contents of cloudflared-config.yml via API.
func (c *Client) GetCloudflaredConfig() (string, error) {
	var result struct {
		Content string `json:"content"`
	}
	err := c.request("GET", "/api/v2/config/cloudflared", nil, &result)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// UpdateCloudflaredConfig updates the contents of cloudflared-config.yml via API.
func (c *Client) UpdateCloudflaredConfig(content string) error {
	body := map[string]string{"content": content}
	return c.request("POST", "/api/v2/config/cloudflared", body, nil)
}

// GetEnvConfigRaw fetches the raw .env file contents via API.
func (c *Client) GetEnvConfigRaw() (string, error) {
	var result struct {
		Content string `json:"content"`
	}
	err := c.request("GET", "/api/v2/config/env", nil, &result)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// UpdateEnvConfig writes updated .env content via API.
func (c *Client) UpdateEnvConfig(content string) error {
	body := map[string]string{"content": content}
	return c.request("POST", "/api/v2/config/env", body, nil)
}

// GetStatus checks the detailed system components health check report.
func (c *Client) GetStatus() (*models.Status, error) {
	var status models.Status
	err := c.request("GET", "/api/v2/system/health", nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// GetStacks returns all discovered stacks merged with DB status.
func (c *Client) GetStacks() ([]models.Stack, error) {
	var stacks []models.Stack
	err := c.request("GET", "/api/v2/stacks", nil, &stacks)
	if err != nil {
		return nil, err
	}
	return stacks, nil
}

// ScanStacks triggers a manual scan of compose files.
func (c *Client) ScanStacks() ([]models.Stack, error) {
	var result struct {
		Stacks []models.Stack `json:"stacks"`
	}
	err := c.request("POST", "/api/v2/stacks/scan", nil, &result)
	if err != nil {
		return nil, err
	}
	return result.Stacks, nil
}

// StartStack deploys a stack by name.
func (c *Client) StartStack(name string) (any, error) {
	var result any
	err := c.request("POST", fmt.Sprintf("/api/v2/stacks/%s/up", name), nil, &result)
	return result, err
}

// StopStack tears down/stops a stack by name.
func (c *Client) StopStack(name string) (any, error) {
	var result any
	err := c.request("POST", fmt.Sprintf("/api/v2/stacks/%s/down", name), nil, &result)
	return result, err
}

// RestartStack restarts a stack by name (stops then starts).
func (c *Client) RestartStack(name string) (any, error) {
	_, err := c.StopStack(name)
	if err != nil {
		return nil, fmt.Errorf("restart failed at stop: %w", err)
	}
	return c.StartStack(name)
}

// GetContainers returns the list of all active containers.
func (c *Client) GetContainers() ([]models.Container, error) {
	var containers []models.Container
	err := c.request("GET", "/api/v2/services", nil, &containers)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

// GetLogs returns the logs (last 100 lines) of a container.
func (c *Client) GetLogs(containerID string) (string, error) {
	var logs models.LogEntry
	err := c.request("GET", fmt.Sprintf("/api/v2/containers/%s/logs?tail=100", containerID), nil, &logs)
	if err != nil {
		return "", err
	}
	return logs.Logs, nil
}

// GetVPNStatus checks the current status of the VPN connection.
func (c *Client) GetVPNStatus() (*models.VPNStatus, error) {
	var status models.VPNStatus
	err := c.request("GET", "/api/v2/vpn/status", nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// ControlVPN sends a start, stop, or restart command to the VPN.
func (c *Client) ControlVPN(action string) (any, error) {
	var result any
	body := map[string]string{"action": action}
	err := c.request("POST", "/api/v2/vpn/control", body, &result)
	return result, err
}

// SwitchVPNRegion switches the VPN connection region.
func (c *Client) SwitchVPNRegion(region string) (any, error) {
	var result any
	body := map[string]string{"region": region}
	err := c.request("POST", "/api/v2/vpn/region", body, &result)
	return result, err
}

// SyncVPNPort triggers manual port synchronization for Gluetun.
func (c *Client) SyncVPNPort() (int, error) {
	var result struct {
		Status        string `json:"status"`
		ForwardedPort int    `json:"forwarded_port"`
	}
	err := c.request("POST", "/api/v2/vpn/sync-port", nil, &result)
	if err != nil {
		return 0, err
	}
	return result.ForwardedPort, nil
}

// CheckVPNLeak runs leak detection on the host and VPN outbound IPs.
func (c *Client) CheckVPNLeak() (*models.VPNLeakReport, error) {
	var report models.VPNLeakReport
	err := c.request("GET", "/api/v2/vpn/check-leak", nil, &report)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// ControlContainer starts, stops, or restarts a container via the API.
func (c *Client) ControlContainer(name, action string) error {
	return c.request("POST", fmt.Sprintf("/api/v2/containers/%s/%s", name, action), nil, nil)
}

// GetQueue returns the list of all queued and finished tasks.
func (c *Client) GetQueue() ([]models.JobRecord, error) {
	var queueList []models.JobRecord
	err := c.request("GET", "/api/v2/queue", nil, &queueList)
	if err != nil {
		return nil, err
	}
	return queueList, nil
}

// CancelQueueJob aborts a pending or active job by ID.
func (c *Client) CancelQueueJob(id string) error {
	body := map[string]string{"id": id}
	return c.request("POST", "/api/v2/queue/cancel", body, nil)
}

// GetAIModels returns the list of available AI models from the Ollama backend.
func (c *Client) GetAIModels() ([]string, error) {
	var modelsList []string
	err := c.request("GET", "/api/v2/ai/models", nil, &modelsList)
	if err != nil {
		return nil, err
	}
	return modelsList, nil
}

// RunAI executes an AI prompt inference request synchronously.
func (c *Client) RunAI(prompt, mode, priority, model string) (string, error) {
	var result struct {
		Response string `json:"response"`
		Model    string `json:"model"`
	}
	body := map[string]string{
		"prompt":   prompt,
		"mode":     mode,
		"priority": priority,
		"model":    model,
	}
	err := c.request("POST", "/api/v2/ai/run", body, &result)
	if err != nil {
		return "", err
	}
	return result.Response, nil
}

// GetPlugins returns all loaded plugins across all kinds.
func (c *Client) GetPlugins() (*models.PluginsResponse, error) {
	var response models.PluginsResponse
	err := c.request("GET", "/api/v2/plugins", nil, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetPluginCatalog correlates remote catalog items with local loaded plugins.
func (c *Client) GetPluginCatalog() ([]models.CatalogItemStatus, error) {
	var catalog []models.CatalogItemStatus
	err := c.request("GET", "/api/v2/plugins/catalog", nil, &catalog)
	if err != nil {
		return nil, err
	}
	return catalog, nil
}

// EnablePlugin enables a plugin by name and kind.
func (c *Client) EnablePlugin(name, kind string) error {
	body := map[string]string{"name": name, "kind": kind}
	return c.request("POST", "/api/v2/plugins/enable", body, nil)
}

// DisablePlugin disables a plugin by name and kind.
func (c *Client) DisablePlugin(name, kind string) error {
	body := map[string]string{"name": name, "kind": kind}
	return c.request("POST", "/api/v2/plugins/disable", body, nil)
}

// InstallPlugin downloads and installs a plugin by name and kind.
func (c *Client) InstallPlugin(name, kind string) error {
	body := map[string]string{"name": name, "kind": kind}
	return c.request("POST", "/api/v2/plugins/install", body, nil)
}

// UninstallPlugin uninstalls a plugin by name and kind.
func (c *Client) UninstallPlugin(name, kind string) error {
	body := map[string]string{"name": name, "kind": kind}
	return c.request("POST", "/api/v2/plugins/uninstall", body, nil)
}

// GetTrayStats returns the host system metrics response for the tray.
func (c *Client) GetTrayStats() (*models.DetailedStats, error) {
	var metrics models.DetailedStats
	err := c.request("GET", "/api/v2/tray/stats", nil, &metrics)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// GetTrayContainers returns all active containers for the tray.
func (c *Client) GetTrayContainers() ([]models.Container, error) {
	var containersList []models.Container
	err := c.request("GET", "/api/v2/tray/containers", nil, &containersList)
	if err != nil {
		return nil, err
	}
	return containersList, nil
}

// GetDoctor runs pre-flight diagnostics on the backend.
func (c *Client) GetDoctor() ([]models.CheckResult, error) {
	var results []models.CheckResult
	err := c.request("GET", "/api/v2/system/doctor", nil, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateDashpass updates dashboard credentials on the daemon.
func (c *Client) UpdateDashpass(username, password string) error {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	return c.request("POST", "/api/v2/auth/dashpass", body, nil)
}

// InitializeStacks setups plugin directories and validates storage path.
func (c *Client) InitializeStacks() (map[string]any, error) {
	var result map[string]any
	err := c.request("POST", "/api/v2/stacks/init", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetDoctorContainers scans container health states via API.
func (c *Client) GetDoctorContainers() ([]models.ContainerResult, error) {
	var results []models.ContainerResult
	err := c.request("GET", "/api/v2/doctor/containers", nil, &results)
	return results, err
}

// GetDoctorMounts validates container volume and bind-mount paths via API.
func (c *Client) GetDoctorMounts() ([]models.MountResult, error) {
	var results []models.MountResult
	err := c.request("GET", "/api/v2/doctor/mounts", nil, &results)
	return results, err
}

// GetDoctorPorts detects port conflicts via API.
func (c *Client) GetDoctorPorts() ([]models.PortResult, error) {
	var results []models.PortResult
	err := c.request("GET", "/api/v2/doctor/ports", nil, &results)
	return results, err
}

// HandleDoctorFix previews or applies automated fixes via API.
func (c *Client) HandleDoctorFix(apply bool, name string) (map[string]any, error) {
	var result map[string]any
	body := map[string]any{"apply": apply, "name": name}
	err := c.request("POST", "/api/v2/doctor/fix", body, &result)
	return result, err
}

// GetDoctorReport generates a full system health report via API.
func (c *Client) GetDoctorReport() (models.DoctorReport, error) {
	var report models.DoctorReport
	err := c.request("GET", "/api/v2/doctor/report", nil, &report)
	return report, err
}

// PullStacks triggers image pull via the API.
func (c *Client) PullStacks(name string) (any, error) {
	var result any
	body := map[string]string{"name": name}
	err := c.request("POST", "/api/v2/stacks/pull", body, &result)
	return result, err
}

// GetStackLogs returns logs (last tail lines) of a stack.
func (c *Client) GetStackLogs(name string, tail int) (string, error) {
	var result struct {
		Logs string `json:"logs"`
	}
	err := c.request("GET", fmt.Sprintf("/api/v2/stacks/%s/logs?tail=%d", name, tail), nil, &result)
	if err != nil {
		return "", err
	}
	return result.Logs, nil
}

// ValidatePlugin validates a plugin manifest YAML.
func (c *Client) ValidatePlugin(yamlContent string) (map[string]any, error) {
	var result map[string]any
	body := map[string]string{"yaml": yamlContent}
	err := c.request("POST", "/api/v2/plugins/validate", body, &result)
	return result, err
}

// MatchPlugin matches a service to a route plugin.
func (c *Client) MatchPlugin(service, image string, labels map[string]string) (map[string]any, error) {
	var result map[string]any
	body := map[string]any{
		"service": service,
		"image":   image,
		"labels":  labels,
	}
	err := c.request("POST", "/api/v2/plugins/match", body, &result)
	return result, err
}

// InstallStackPlugin parameterizes and installs a stack plugin.
func (c *Client) InstallStackPlugin(name string, env map[string]string) (map[string]any, error) {
	var result map[string]any
	body := map[string]any{
		"name": name,
		"env":  env,
	}
	err := c.request("POST", "/api/v2/plugins/install-stack", body, &result)
	return result, err
}

// SyncPlugins synchronizes Traefik dynamic provider configuration.
func (c *Client) SyncPlugins() (any, error) {
	var result any
	err := c.request("POST", "/api/v2/plugins/sync", nil, &result)
	return result, err
}

// ReconcileSystem triggers the state reconciliation loop on the daemon.
func (c *Client) ReconcileSystem() (map[string]any, error) {
	var result map[string]any
	err := c.request("POST", "/api/v2/system/reconcile", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetSystemServices retrieves the systemd services list.
func (c *Client) GetSystemServices() ([]models.ServiceStatus, error) {
	var list []models.ServiceStatus
	err := c.request("GET", "/api/v2/system/services", nil, &list)
	return list, err
}

// ControlSystemService controls systemd services.
func (c *Client) ControlSystemService(name, action string) error {
	return c.request("POST", fmt.Sprintf("/api/v2/system/services/%s/%s", name, action), nil, nil)
}

// StorageResponse holds disk partitions and samba shares
type StorageResponse struct {
	Partitions []models.DiskPartition `json:"partitions"`
	Samba      []models.SambaShare    `json:"samba"`
}

// GetSystemStorage retrieves partition and Samba info.
func (c *Client) GetSystemStorage() (*StorageResponse, error) {
	var resp StorageResponse
	err := c.request("GET", "/api/v2/system/storage", nil, &resp)
	return &resp, err
}

// GetCronJobs retrieves scheduled cron and systemd timer jobs.
func (c *Client) GetCronJobs() ([]models.CronJob, error) {
	var list []models.CronJob
	err := c.request("GET", "/api/v2/system/cron", nil, &list)
	return list, err
}

// GetSystemUpdates retrieves available upgrades.
func (c *Client) GetSystemUpdates() (*models.SystemUpdates, error) {
	var updates models.SystemUpdates
	err := c.request("GET", "/api/v2/system/updates", nil, &updates)
	return &updates, err
}

// ApplySystemUpdates triggers system package upgrades.
func (c *Client) ApplySystemUpdates() error {
	return c.request("POST", "/api/v2/system/updates/apply", nil, nil)
}

// GetDockerImages lists Docker images.
func (c *Client) GetDockerImages() ([]models.DockerImage, error) {
	var list []models.DockerImage
	err := c.request("GET", "/api/v2/docker/images", nil, &list)
	return list, err
}

// PruneDockerImages prunes unused images and returns space saved.
func (c *Client) PruneDockerImages() (int64, error) {
	var res struct {
		Reclaimed int64 `json:"reclaimed"`
	}
	err := c.request("POST", "/api/v2/docker/images/prune", nil, &res)
	return res.Reclaimed, err
}

// GetDockerVolumes lists volumes.
func (c *Client) GetDockerVolumes() ([]models.DockerVolume, error) {
	var list []models.DockerVolume
	err := c.request("GET", "/api/v2/docker/volumes", nil, &list)
	return list, err
}

// PruneDockerVolumes prunes unused volumes and returns space saved.
func (c *Client) PruneDockerVolumes() (int64, error) {
	var res struct {
		Reclaimed int64 `json:"reclaimed"`
	}
	err := c.request("POST", "/api/v2/docker/volumes/prune", nil, &res)
	return res.Reclaimed, err
}

// GetDockerNetworks lists networks.
func (c *Client) GetDockerNetworks() ([]models.DockerNetwork, error) {
	var list []models.DockerNetwork
	err := c.request("GET", "/api/v2/docker/networks", nil, &list)
	return list, err
}

// PruneDockerNetworks prunes unused networks and returns count.
func (c *Client) PruneDockerNetworks() (int, error) {
	var res struct {
		PrunedCount int `json:"pruned_count"`
	}
	err := c.request("POST", "/api/v2/docker/networks/prune", nil, &res)
	return res.PrunedCount, err
}
