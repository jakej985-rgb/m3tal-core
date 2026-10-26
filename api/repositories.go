package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)

// RepoConfig represents an APT repository configured for M3TAL plugins.
type RepoConfig struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Distribution string    `json:"distribution"`
	Component    string    `json:"component"`
	KeyringPath  string    `json:"keyring_path,omitempty"`
	KeyURL       string    `json:"key_url,omitempty"`
	Channel      string    `json:"channel"` // "stable" or "beta"
	Enabled      bool      `json:"enabled"`
	IsOfficial   bool      `json:"is_official"`
	SourcesPath  string    `json:"sources_path,omitempty"`
	KeyringValid bool      `json:"keyring_valid"`
	LastSynced   time.Time `json:"last_synced,omitempty"`
}

func getRepositoriesFile() string {
	return filepath.Join(getSettingsDir(), "repositories.json")
}

func loadCustomRepositories() []RepoConfig {
	file := getRepositoriesFile()
	data, err := os.ReadFile(file)
	if err != nil {
		return []RepoConfig{}
	}

	var repos []RepoConfig
	if err := json.Unmarshal(data, &repos); err != nil {
		return []RepoConfig{}
	}
	return repos
}

func saveCustomRepositories(repos []RepoConfig) error {
	dir := getSettingsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getRepositoriesFile(), data, 0644)
}

func getOfficialRepository() RepoConfig {
	sourcesPath := "/etc/apt/sources.list.d/m3tal.list"
	keyringPath := "/etc/apt/keyrings/m3tal-archive-keyring.gpg"

	keyringValid := false
	if fi, err := os.Stat(keyringPath); err == nil && fi.Size() > 0 {
		keyringValid = true
	}

	enabled := false
	if fi, err := os.Stat(sourcesPath); err == nil && fi.Size() > 0 {
		enabled = true
	}

	return RepoConfig{
		ID:           "official-m3tal",
		Name:         "Official M3TAL Keyring Repository",
		URL:          "https://jakej985-rgb.github.io/m3tal-apt-key",
		Distribution: "stable",
		Component:    "main",
		KeyringPath:  keyringPath,
		KeyURL:       "https://jakej985-rgb.github.io/m3tal-apt-key/m3tal-archive-keyring.gpg",
		Channel:      "stable",
		Enabled:      enabled,
		IsOfficial:   true,
		SourcesPath:  sourcesPath,
		KeyringValid: keyringValid,
		LastSynced:   time.Now(),
	}
}

// GetRepositories returns configured repositories and discovered packages.
// GET /api/v2/repositories
func (h *PluginHandlers) GetRepositories(w http.ResponseWriter, r *http.Request) {
	official := getOfficialRepository()
	custom := loadCustomRepositories()

	pkgs, _ := DiscoverAptPlugins(r.Context(), false)

	sendSuccess(w, http.StatusOK, map[string]any{
		"official":     official,
		"custom":       custom,
		"packages":     pkgs,
		"last_checked": time.Now().Unix(),
	}, nil)
}

// SaveRepository adds or updates a repository.
// POST /api/v2/repositories
func (h *PluginHandlers) SaveRepository(w http.ResponseWriter, r *http.Request) {
	var req RepoConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_BODY", "failed to parse repository configuration", nil)
		return
	}

	if req.URL == "" {
		sendError(w, http.StatusBadRequest, "INVALID_URL", "repository URL is required", nil)
		return
	}

	if req.Distribution == "" {
		req.Distribution = "stable"
	}
	if req.Component == "" {
		req.Component = "main"
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("repo-%d", time.Now().UnixNano())
	}

	custom := loadCustomRepositories()
	updated := false
	for i, existing := range custom {
		if existing.ID == req.ID || existing.URL == req.URL {
			custom[i] = req
			updated = true
			break
		}
	}
	if !updated {
		custom = append(custom, req)
	}

	if err := saveCustomRepositories(custom); err != nil {
		sendError(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil)
		return
	}

	InvalidateAptCache()

	sendSuccess(w, http.StatusOK, map[string]any{
		"message":    "Repository saved successfully",
		"repository": req,
		"custom":     custom,
	}, nil)
}

// DeleteRepository removes a custom repository.
// DELETE /api/v2/repositories/{id}
func (h *PluginHandlers) DeleteRepository(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		sendError(w, http.StatusBadRequest, "INVALID_ID", "repository id required", nil)
		return
	}

	custom := loadCustomRepositories()
	var next []RepoConfig
	found := false
	for _, repo := range custom {
		if repo.ID == id {
			found = true
			continue
		}
		next = append(next, repo)
	}

	if !found {
		sendError(w, http.StatusNotFound, "REPO_NOT_FOUND", "repository not found", nil)
		return
	}

	if err := saveCustomRepositories(next); err != nil {
		sendError(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error(), nil)
		return
	}

	InvalidateAptCache()

	sendSuccess(w, http.StatusOK, map[string]any{
		"message": "Repository removed successfully",
		"custom":  next,
	}, nil)
}

// RefreshRepositories invalidates cached policy and performs live APT candidate re-query.
// POST /api/v2/repositories/refresh
func (h *PluginHandlers) RefreshRepositories(w http.ResponseWriter, r *http.Request) {
	InvalidateAptCache()

	pkgs, err := DiscoverAptPlugins(r.Context(), true)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "REFRESH_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"message":      "APT repository cache refreshed successfully",
		"packages":     pkgs,
		"last_checked": time.Now().Unix(),
	}, nil)
}
