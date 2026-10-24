package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
	"strings"
)

// PluginState represents runtime state of a plugin subprocess.
type PluginState string

const (
	StateStopped  PluginState = "stopped"
	StateStarting PluginState = "starting"
	StateRunning  PluginState = "running"
	StateDegraded PluginState = "degraded"
	StateCrashed  PluginState = "crashed"
)

// PluginInstance represents a managed running plugin subprocess.
type PluginInstance struct {
	Manifest     PluginManifest
	State        PluginState
	Capabilities []Capability
	Tabs         []PluginTabSpec
	Actions      []ActionSpec

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader

	reqID  atomic.Uint64
	mu     sync.Mutex
	cancel context.CancelFunc

	LastCollectedAt   time.Time
	LastResult        *CollectResult
	LastError         string
	ExplicitlyStopped bool
}

// Manager supervises discovery, lifecycle, and communication for all plugins.
type Manager struct {
	mu           sync.RWMutex
	plugins      map[string]*PluginInstance
	searchPaths  []string
	coreVersion  string
	pollInterval time.Duration
	cancel       context.CancelFunc
}

// NewManager creates a new plugin manager with standard search paths.
func NewManager(coreVersion string, customPaths ...string) *Manager {
	var paths []string
	if len(customPaths) > 0 {
		paths = customPaths
	} else {
		home, _ := os.UserHomeDir()
		paths = []string{
			"./plugins",
		}
		if home != "" {
			paths = append(paths, filepath.Join(home, ".config", "m3tal", "plugins"))
		}
		paths = append(paths, "/var/lib/m3tal/plugins", "/usr/lib/m3tal/plugins", "/etc/m3tal/plugins")
	}

	return &Manager{
		plugins:      make(map[string]*PluginInstance),
		searchPaths:  paths,
		coreVersion:  coreVersion,
		pollInterval: 2 * time.Second,
	}
}

// Discover scans search paths for valid plugin.yaml manifests.
func (m *Manager) Discover() ([]PluginManifest, error) {
	var discovered []PluginManifest
	seen := make(map[string]bool)

	for _, dir := range m.searchPaths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // directory may not exist, skip silently
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginDir := filepath.Join(dir, entry.Name())
			manifestPath := filepath.Join(pluginDir, "plugin.yaml")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				manifestPath = filepath.Join(pluginDir, "plugin.yml")
				if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
					continue
				}
			}

			data, err := os.ReadFile(manifestPath)
			if err != nil {
				log.Printf("[plugins] Failed to read %s: %v", manifestPath, err)
				continue
			}

			var manifest PluginManifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				log.Printf("[plugins] Failed to parse %s: %v", manifestPath, err)
				continue
			}

			manifest.Dir = pluginDir
			name := manifest.Metadata.Name
			if name == "" {
				name = entry.Name()
				manifest.Metadata.Name = name
			}

			if !seen[name] {
				seen[name] = true
				discovered = append(discovered, manifest)

				m.mu.Lock()
				if inst, exists := m.plugins[name]; exists {
					if inst.State == StateStopped || inst.State == StateCrashed {
						inst.Manifest = manifest
					}
				} else {
					m.plugins[name] = &PluginInstance{
						Manifest: manifest,
						State:    StateStopped,
					}
				}
				m.mu.Unlock()
			}
		}
	}

	return discovered, nil
}

// StartAutoDiscovery launches a background routine that periodically senses
// newly installed plugins across all search paths and automatically starts them.
func (m *Manager) StartAutoDiscovery(ctx context.Context, interval time.Duration) {
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return
	}
	subCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.mu.Unlock()

	// Initial synchronous discovery and auto-start
	_, _ = m.DiscoverAndAutoStart(subCtx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-subCtx.Done():
				return
			case <-ticker.C:
				_, _ = m.DiscoverAndAutoStart(subCtx)
			}
		}
	}()
}

// DiscoverAndAutoStart scans all search paths and automatically starts any newly
// discovered plugins that have not been explicitly stopped by the operator.
func (m *Manager) DiscoverAndAutoStart(ctx context.Context) ([]PluginManifest, error) {
	discovered, err := m.Discover()
	if err != nil {
		return nil, err
	}

	for _, manifest := range discovered {
		name := manifest.Metadata.Name
		m.mu.Lock()
		inst, exists := m.plugins[name]
		if !exists {
			inst = &PluginInstance{
				Manifest: manifest,
				State:    StateStopped,
			}
			m.plugins[name] = inst
		}
		m.mu.Unlock()

		if inst.State == StateStopped && !inst.ExplicitlyStopped {
			if err := inst.Start(ctx, m.coreVersion); err != nil {
				log.Printf("[plugins] Auto-start skipped or failed for %q: %v", name, err)
			} else {
				log.Printf("[plugins] Auto-started plugin %q", name)
			}
		}
	}

	return discovered, nil
}

// LoadAndStart starts all discovered plugins.
func (m *Manager) LoadAndStart(ctx context.Context) error {
	_, err := m.DiscoverAndAutoStart(ctx)
	return err
}

// Start spawns the plugin subprocess and performs the m3tal.init handshake.
func (p *PluginInstance) Start(ctx context.Context, coreVersion string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State == StateRunning {
		return nil
	}

	cmdPath := p.Manifest.Spec.Command
	if !filepath.IsAbs(cmdPath) {
		cmdPath = filepath.Join(p.Manifest.Dir, cmdPath)
	}

	if _, err := os.Stat(cmdPath); err != nil {
		p.State = StateCrashed
		p.LastError = fmt.Sprintf("Executable not found: %s", cmdPath)
		return errors.New(p.LastError)
	}

	// Canonical process naming: all plugins follow m3tal-<process>
	procName := p.Manifest.Spec.ProcessName
	if procName == "" {
		procName = p.Manifest.Metadata.Name
	}
	if !strings.HasPrefix(procName, "m3tal-") {
		procName = "m3tal-" + procName
	}

	if absCmdPath, err := filepath.Abs(cmdPath); err == nil {
		cmdPath = absCmdPath
	}

	// 1. Direct Execution: If the binary is already canonically named (e.g. m3tal-docker),
	// execute it directly from its install location without creating any symlink.
	// 2. Third-Party / Non-Canonical: Create a canonical symlink in the secure daemon runtime
	// directory ($RUNTIME_DIRECTORY or /run/m3tal/plugins-bin) or user cache rather than /tmp.
	if filepath.Base(cmdPath) != procName {
		binDir := ""
		if rtDir := os.Getenv("RUNTIME_DIRECTORY"); rtDir != "" {
			binDir = filepath.Join(rtDir, "plugins-bin")
		} else if _, err := os.Stat("/run/m3tal"); err == nil && os.Geteuid() == 0 {
			binDir = "/run/m3tal/plugins-bin"
		} else if xdgRt := os.Getenv("XDG_RUNTIME_DIR"); xdgRt != "" {
			binDir = filepath.Join(xdgRt, "m3tal", "plugins-bin")
		} else if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
			binDir = filepath.Join(cacheDir, "m3tal", "plugins-bin")
		} else {
			binDir = filepath.Join(os.TempDir(), "m3tal-plugins-bin")
		}
		_ = os.MkdirAll(binDir, 0755)
		canonicalExec := filepath.Join(binDir, procName)
		_ = os.Remove(canonicalExec)
		if err := os.Symlink(cmdPath, canonicalExec); err == nil {
			cmdPath = canonicalExec
		}
	}

	pCtx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	cmd := exec.CommandContext(pCtx, cmdPath, p.Manifest.Spec.Args...)
	cmd.Dir = p.Manifest.Dir
	cmd.Env = append(os.Environ(),
		"M3TAL_CORE_VERSION="+coreVersion,
		"M3TAL_PLUGIN_DIR="+p.Manifest.Dir,
		"M3TAL_PROCESS_NAME="+procName,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		p.State = StateCrashed
		return fmt.Errorf("stdin pipe error: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		cancel()
		p.State = StateCrashed
		return fmt.Errorf("stdout pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		cancel()
		p.State = StateCrashed
		p.LastError = err.Error()
		return fmt.Errorf("failed to start process: %w", err)
	}

	p.cmd = cmd
	p.stdin = stdin
	p.reader = bufio.NewReader(stdout)
	p.State = StateStarting

	// Perform m3tal.init handshake
	initParams := InitParams{
		CoreVersion: coreVersion,
		Config:      p.Manifest.Config,
	}

	var initRes InitResult
	if err := p.callLocked(ctx, MethodInit, initParams, &initRes); err != nil {
		p.State = StateDegraded
		p.LastError = fmt.Sprintf("Handshake failed: %v", err)
		log.Printf("[plugins] Handshake error for %s: %v", p.Manifest.Metadata.Name, err)
		return err
	}

	p.Capabilities = initRes.Capabilities
	p.Tabs = initRes.Tabs
	p.Actions = initRes.Actions
	p.State = StateRunning

	// Monitor process termination in background
	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		if p.State == StateRunning || p.State == StateStarting {
			p.State = StateCrashed
			p.LastError = "Process exited unexpectedly"
		}
		p.mu.Unlock()
	}()

	return nil
}

// Stop cleanly terminates the plugin process.
func (p *PluginInstance) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State == StateStopped || p.cmd == nil {
		return nil
	}

	p.State = StateStopped
	if p.stdin != nil {
		_ = p.stdin.Close()
	}
	if p.cancel != nil {
		p.cancel()
	}

	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(os.Interrupt)
		time.Sleep(100 * time.Millisecond)
		_ = p.cmd.Process.Kill()
	}

	return nil
}

// Collect calls m3tal.collect on the plugin to fetch latest metrics and tables.
func (p *PluginInstance) Collect(ctx context.Context) (*CollectResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != StateRunning {
		return nil, fmt.Errorf("plugin %s not running (state=%s)", p.Manifest.Metadata.Name, p.State)
	}

	params := CollectParams{
		Timestamp: time.Now().Unix(),
	}

	var res CollectResult
	err := p.callLocked(ctx, MethodCollect, params, &res)
	if err != nil {
		p.LastError = err.Error()
		return nil, err
	}

	p.LastCollectedAt = time.Now()
	p.LastResult = &res
	return &res, nil
}

// Render calls m3tal.render to get UI view for a given tab.
func (p *PluginInstance) Render(ctx context.Context, tabID string, width, height int, theme string) (*RenderResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != StateRunning {
		return nil, fmt.Errorf("plugin %s not running", p.Manifest.Metadata.Name)
	}

	params := RenderParams{
		TabID:  tabID,
		Width:  width,
		Height: height,
		Theme:  theme,
	}

	var res RenderResult
	err := p.callLocked(ctx, MethodRender, params, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Action executes an action hook on the plugin.
func (p *PluginInstance) Action(ctx context.Context, actionID, target string, args map[string]any) (*ActionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != StateRunning {
		return nil, fmt.Errorf("plugin %s not running", p.Manifest.Metadata.Name)
	}

	params := ActionParams{
		ActionID: actionID,
		Target:   target,
		Args:     args,
	}

	var res ActionResult
	err := p.callLocked(ctx, MethodAction, params, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// callLocked sends a JSON-RPC 2.0 message over stdin and reads response from stdout.
func (p *PluginInstance) callLocked(ctx context.Context, method string, params any, out any) error {
	id := p.reqID.Add(1)

	var paramBytes json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params error: %w", err)
		}
		paramBytes = b
	}

	req := RPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramBytes,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request error: %w", err)
	}
	reqData = append(reqData, '\n')

	if _, err := p.stdin.Write(reqData); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	// Read response line with timeout
	type readResult struct {
		line []byte
		err  error
	}

	ch := make(chan readResult, 1)
	go func() {
		line, err := p.reader.ReadBytes('\n')
		ch <- readResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return fmt.Errorf("read response error: %w", res.err)
		}

		var resp RPCResponse
		if err := json.Unmarshal(res.line, &resp); err != nil {
			return fmt.Errorf("unmarshal response error: %w (raw: %s)", err, string(res.line))
		}

		if resp.Error != nil {
			return resp.Error
		}

		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("unmarshal result payload error: %w", err)
			}
		}

		return nil
	}
}

// StopAll stops all active plugins and terminates background auto-discovery.
func (m *Manager) StopAll() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	pluginsCopy := make([]*PluginInstance, 0, len(m.plugins))
	for _, p := range m.plugins {
		pluginsCopy = append(pluginsCopy, p)
	}
	m.mu.Unlock()

	for _, p := range pluginsCopy {
		_ = p.Stop()
	}
}

// GetAllPlugins returns map of managed plugins.
func (m *Manager) GetAllPlugins() []*PluginInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*PluginInstance, 0, len(m.plugins))
	for _, p := range m.plugins {
		list = append(list, p)
	}
	return list
}

// GetAllTabs gathers all UI tabs registered across running plugins.
func (m *Manager) GetAllTabs() []PluginTabSpec {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tabs []PluginTabSpec
	for _, p := range m.plugins {
		if p.State == StateRunning {
			if len(p.Tabs) > 0 {
				for _, tab := range p.Tabs {
					tabCopy := tab
					if tabCopy.Plugin == "" {
						tabCopy.Plugin = p.Manifest.Metadata.Name
					}
					if tabCopy.URL == "" {
						tabCopy.URL = "/plugins/" + p.Manifest.Metadata.Name + "/"
					}
					if tabCopy.Category == "" {
						if p.Manifest.Spec.UI.Category != "" {
							tabCopy.Category = p.Manifest.Spec.UI.Category
						} else if p.Manifest.Spec.UI.Route != "" {
							tabCopy.Category = strings.TrimPrefix(p.Manifest.Spec.UI.Route, "/")
						} else {
							tabCopy.Category = tabCopy.Plugin
						}
					}
					if tabCopy.CategoryTitle == "" {
						if p.Manifest.Spec.UI.CategoryTitle != "" {
							tabCopy.CategoryTitle = p.Manifest.Spec.UI.CategoryTitle
						} else if p.Manifest.Spec.UI.Title != "" {
							tabCopy.CategoryTitle = p.Manifest.Spec.UI.Title
						}
					}
					tabs = append(tabs, tabCopy)
				}
			} else if p.Manifest.HasCapability(CapabilityUIView) {
				title := p.Manifest.Spec.UI.Title
				if title == "" {
					title = p.Manifest.Metadata.Name
				}
				category := p.Manifest.Spec.UI.Category
				if category == "" && p.Manifest.Spec.UI.Route != "" {
					category = strings.TrimPrefix(p.Manifest.Spec.UI.Route, "/")
				}
				if category == "" {
					category = p.Manifest.Metadata.Name
				}
				categoryTitle := p.Manifest.Spec.UI.CategoryTitle
				if categoryTitle == "" {
					categoryTitle = title
				}
				tabs = append(tabs, PluginTabSpec{
					ID:            p.Manifest.Metadata.Name + "_view",
					Title:         title,
					Category:      category,
					CategoryTitle: categoryTitle,
					Order:         p.Manifest.Spec.UI.Order,
					Plugin:        p.Manifest.Metadata.Name,
					URL:           "/plugins/" + p.Manifest.Metadata.Name + "/",
				})
			}
		}
	}
	return tabs
}

// GetPlugin returns a managed plugin instance by name.
func (m *Manager) GetPlugin(name string) *PluginInstance {
	m.mu.RLock()
	p, exists := m.plugins[name]
	if !exists {
		for _, inst := range m.plugins {
			if inst.Manifest.Spec.ProcessName == name ||
				inst.Manifest.Metadata.Name == strings.TrimPrefix(name, "m3tal-") ||
				"m3tal-"+inst.Manifest.Metadata.Name == name {
				p = inst
				break
			}
		}
	}
	m.mu.RUnlock()
	return p
}

// StartPlugin starts a named plugin.
func (m *Manager) StartPlugin(ctx context.Context, name string) error {
	m.mu.Lock()
	p, exists := m.plugins[name]
	if !exists {
		for _, inst := range m.plugins {
			if inst.Manifest.Spec.ProcessName == name ||
				inst.Manifest.Metadata.Name == strings.TrimPrefix(name, "m3tal-") ||
				"m3tal-"+inst.Manifest.Metadata.Name == name {
				p = inst
				exists = true
				break
			}
		}
	}
	m.mu.Unlock()

	if !exists {
		// Run discovery to refresh search paths if newly installed
		if _, err := m.Discover(); err == nil {
			m.mu.Lock()
			p, exists = m.plugins[name]
			if !exists {
				for _, inst := range m.plugins {
					if inst.Manifest.Spec.ProcessName == name ||
						inst.Manifest.Metadata.Name == strings.TrimPrefix(name, "m3tal-") ||
						"m3tal-"+inst.Manifest.Metadata.Name == name {
						p = inst
						exists = true
						break
					}
				}
			}
			m.mu.Unlock()
		}
	}

	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}
	p.ExplicitlyStopped = false
	return p.Start(ctx, m.coreVersion)
}

// StopPlugin stops a named plugin.
func (m *Manager) StopPlugin(name string) error {
	m.mu.RLock()
	p, exists := m.plugins[name]
	if !exists {
		for _, inst := range m.plugins {
			if inst.Manifest.Spec.ProcessName == name ||
				inst.Manifest.Metadata.Name == strings.TrimPrefix(name, "m3tal-") ||
				"m3tal-"+inst.Manifest.Metadata.Name == name {
				p = inst
				exists = true
				break
			}
		}
	}
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}
	p.ExplicitlyStopped = true
	return p.Stop()
}
