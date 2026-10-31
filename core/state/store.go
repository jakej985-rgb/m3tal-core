package state

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for M3TAL state persistence.
type Store struct {
	db        *sql.DB
	ConfigDir string // Optional directory for api_keys.json export (defaults to ~/.config/m3tal or /etc/m3tal)
}

// APIKeyRecord is a stored API authorization key.
type APIKeyRecord struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Token      string  `json:"token"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

// RouteRecord is a stored route with its database ID.
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

// StackRecord is a stored stack reference.
type StackRecord struct {
	Name         string `json:"name"`
	ComposePath  string `json:"compose_path"`
	LastDeployed string `json:"last_deployed,omitempty"`
	Status       string `json:"status"`
}

// MiddlewareRecord is a stored middleware definition.
type MiddlewareRecord struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Config    map[string]string `json:"config"`
	CreatedAt string            `json:"created_at"`
}

// Open creates or opens a SQLite database at the given path.
// It creates parent directories if they don't exist.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create database directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("cannot open database %s: %w", path, err)
	}

	// Enable WAL mode for concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot set WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot enable foreign keys: %w", err)
	}

	// Set busy timeout for concurrent access
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot set busy timeout: %w", err)
	}

	s := &Store{db: db}
	if err := s.Migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Ping verifies the database connection is alive.
func (s *Store) Ping() error {
	return s.db.Ping()
}

// Migrate creates the required tables if they don't exist.
func (s *Store) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS routes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service TEXT NOT NULL,
			domain TEXT NOT NULL UNIQUE,
			port INTEGER NOT NULL,
			entrypoints TEXT DEFAULT 'web',
			stack TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS stacks (
			name TEXT PRIMARY KEY,
			compose_path TEXT NOT NULL,
			last_deployed DATETIME,
			status TEXT DEFAULT 'unknown'
		)`,
		`CREATE TABLE IF NOT EXISTS middleware (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			type TEXT NOT NULL,
			config TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plugin_states (
			name TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			config TEXT DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used_at DATETIME
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	// Schema migrations for v2 proxy (SSL and Middlewares support)
	_, _ = s.db.Exec("ALTER TABLE routes ADD COLUMN ssl INTEGER DEFAULT 0")
	_, _ = s.db.Exec("ALTER TABLE routes ADD COLUMN middlewares TEXT DEFAULT ''")

	// Ensure at least one primary API key exists if empty
	var keyCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM api_keys").Scan(&keyCount); err == nil && keyCount == 0 {
		_, _ = s.CreateAPIKey("Default Key")
	}

	return nil
}

// --- Routes ---

// CreateRoute inserts a new route and returns its ID.
func (s *Store) CreateRoute(service, domain string, port int, entrypoints, stack string, ssl bool, middlewares string) (int64, error) {
	if entrypoints == "" {
		entrypoints = "web"
	}
	sslVal := 0
	if ssl {
		sslVal = 1
	}
	result, err := s.db.Exec(
		`INSERT INTO routes (service, domain, port, entrypoints, stack, ssl, middlewares) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		service, domain, port, entrypoints, stack, sslVal, middlewares,
	)
	if err != nil {
		return 0, fmt.Errorf("cannot create route: %w", err)
	}
	return result.LastInsertId()
}

// ListRoutes returns all stored routes.
func (s *Store) ListRoutes() ([]RouteRecord, error) {
	rows, err := s.db.Query(`SELECT id, service, domain, port, entrypoints, stack, ssl, COALESCE(middlewares, ''), created_at FROM routes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []RouteRecord
	for rows.Next() {
		var r RouteRecord
		var sslInt int
		if err := rows.Scan(&r.ID, &r.Service, &r.Domain, &r.Port, &r.Entrypoints, &r.Stack, &sslInt, &r.Middlewares, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.SSL = (sslInt == 1)
		routes = append(routes, r)
	}
	return routes, rows.Err()
}

// GetRouteByDomain looks up a route by its domain.
func (s *Store) GetRouteByDomain(domain string) (*RouteRecord, error) {
	var r RouteRecord
	var sslInt int
	err := s.db.QueryRow(
		`SELECT id, service, domain, port, entrypoints, stack, ssl, COALESCE(middlewares, ''), created_at FROM routes WHERE domain = ?`,
		domain,
	).Scan(&r.ID, &r.Service, &r.Domain, &r.Port, &r.Entrypoints, &r.Stack, &sslInt, &r.Middlewares, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.SSL = (sslInt == 1)
	return &r, nil
}

// DeleteRoute removes a route by ID.
func (s *Store) DeleteRoute(id int64) error {
	result, err := s.db.Exec(`DELETE FROM routes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("route %d not found", id)
	}
	return nil
}

// --- Stacks ---

// UpsertStack creates or updates a stack record.
func (s *Store) UpsertStack(name, composePath string) error {
	_, err := s.db.Exec(
		`INSERT INTO stacks (name, compose_path, status) VALUES (?, ?, 'discovered')
		 ON CONFLICT(name) DO UPDATE SET compose_path = excluded.compose_path`,
		name, composePath,
	)
	return err
}

// UpdateStackStatus updates the deployment status and timestamp.
func (s *Store) UpdateStackStatus(name, status string) error {
	_, err := s.db.Exec(
		`UPDATE stacks SET status = ?, last_deployed = ? WHERE name = ?`,
		status, time.Now().UTC().Format(time.RFC3339), name,
	)
	return err
}

// ListStacks returns all stored stacks.
func (s *Store) ListStacks() ([]StackRecord, error) {
	rows, err := s.db.Query(`SELECT name, compose_path, COALESCE(last_deployed, ''), status FROM stacks ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stacks []StackRecord
	for rows.Next() {
		var st StackRecord
		if err := rows.Scan(&st.Name, &st.ComposePath, &st.LastDeployed, &st.Status); err != nil {
			return nil, err
		}
		stacks = append(stacks, st)
	}
	return stacks, rows.Err()
}

// --- Middleware ---

// CreateMiddleware inserts a new middleware definition.
func (s *Store) CreateMiddleware(name, mwType string, config map[string]string) (int64, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}
	result, err := s.db.Exec(
		`INSERT INTO middleware (name, type, config) VALUES (?, ?, ?)`,
		name, mwType, string(configJSON),
	)
	if err != nil {
		return 0, fmt.Errorf("cannot create middleware: %w", err)
	}
	return result.LastInsertId()
}

// ListMiddleware returns all stored middleware definitions.
func (s *Store) ListMiddleware() ([]MiddlewareRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, type, config, created_at FROM middleware ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mws []MiddlewareRecord
	for rows.Next() {
		var m MiddlewareRecord
		var configJSON string
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &configJSON, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Config = make(map[string]string)
		json.Unmarshal([]byte(configJSON), &m.Config)
		mws = append(mws, m)
	}
	return mws, rows.Err()
}

// DeleteMiddleware removes a middleware by ID.
func (s *Store) DeleteMiddleware(id int64) error {
	result, err := s.db.Exec(`DELETE FROM middleware WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("middleware %d not found", id)
	}
	return nil
}

// --- Plugin States ---

// PluginStateRecord represents a persistent record of a plugin's configuration and enablement status.
type PluginStateRecord struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Enabled     bool   `json:"enabled"`
	InstalledAt string `json:"installed_at"`
	Config      string `json:"config"`
}

// SetPluginState creates or updates the status and config of a plugin.
func (s *Store) SetPluginState(name, kind string, enabled bool, config string) error {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	if config == "" {
		config = "{}"
	}
	_, err := s.db.Exec(
		`INSERT INTO plugin_states (name, kind, enabled, config) VALUES (?, ?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET enabled = excluded.enabled, config = excluded.config`,
		name, kind, enabledVal, config,
	)
	if err != nil {
		return fmt.Errorf("cannot set plugin state: %w", err)
	}
	return nil
}

// GetPluginState retrieves a single plugin's state record.
func (s *Store) GetPluginState(name string) (*PluginStateRecord, error) {
	var rec PluginStateRecord
	var enabledInt int
	err := s.db.QueryRow(
		`SELECT name, kind, enabled, installed_at, config FROM plugin_states WHERE name = ?`,
		name,
	).Scan(&rec.Name, &rec.Kind, &enabledInt, &rec.InstalledAt, &rec.Config)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get plugin state: %w", err)
	}
	rec.Enabled = (enabledInt == 1)
	return &rec, nil
}

// DeletePluginState removes a plugin's state record when uninstalled.
func (s *Store) DeletePluginState(name string) error {
	_, err := s.db.Exec(`DELETE FROM plugin_states WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("cannot delete plugin state: %w", err)
	}
	return nil
}

// ListPluginStates returns all stored plugin states.
func (s *Store) ListPluginStates() ([]PluginStateRecord, error) {
	rows, err := s.db.Query(`SELECT name, kind, enabled, installed_at, config FROM plugin_states ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("cannot list plugin states: %w", err)
	}
	defer rows.Close()

	var list []PluginStateRecord
	for rows.Next() {
		var rec PluginStateRecord
		var enabledInt int
		if err := rows.Scan(&rec.Name, &rec.Kind, &enabledInt, &rec.InstalledAt, &rec.Config); err != nil {
			return nil, err
		}
		rec.Enabled = (enabledInt == 1)
		list = append(list, rec)
	}
	return list, rows.Err()
}

// GetStatePath returns the default database path, respecting overrides.
func GetStatePath() string {
	if p := os.Getenv("M3TAL_STATE_DB"); p != "" {
		return p
	}
	// System install path
	sysPath := "/var/lib/m3tal/state.db"
	if dir := filepath.Dir(sysPath); dirExists(dir) {
		return sysPath
	}
	// Fallback to local
	return "./data/state.db"
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// --- API Keys (Jellyfin-style server generated tokens) ---

// ListAPIKeys returns all stored API keys.
func (s *Store) ListAPIKeys() ([]APIKeyRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, token, created_at, last_used_at FROM api_keys ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("cannot list API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKeyRecord
	for rows.Next() {
		var rec APIKeyRecord
		var lastUsed sql.NullString
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Token, &rec.CreatedAt, &lastUsed); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			rec.LastUsedAt = &lastUsed.String
		}
		keys = append(keys, rec)
	}
	if keys == nil {
		keys = []APIKeyRecord{}
	}
	return keys, rows.Err()
}

// CreateAPIKey generates a cryptographically secure random token (like Jellyfin) and stores it with the specified name.
// Custom key entries are prohibited: the token is strictly server-generated.
func (s *Store) CreateAPIKey(name string) (*APIKeyRecord, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("API key name cannot be empty")
	}

	// 16 bytes = 32 hex characters cryptographically secure token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := s.db.Exec(`INSERT INTO api_keys (name, token, created_at) VALUES (?, ?, ?)`, trimmedName, token, now)
	if err != nil {
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	rec := &APIKeyRecord{
		ID:        id,
		Name:      trimmedName,
		Token:     token,
		CreatedAt: now,
	}

	_ = s.SyncAPIKeysToFile()
	return rec, nil
}

// DeleteAPIKey removes an API key by its database ID.
func (s *Store) DeleteAPIKey(id int64) error {
	res, err := s.db.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("cannot delete API key: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("API key with ID %d not found", id)
	}
	_ = s.SyncAPIKeysToFile()
	return nil
}

// ValidateAPIKey checks if a token exists in the store and records its last usage.
func (s *Store) ValidateAPIKey(token string) bool {
	if token == "" {
		return false
	}
	var id int64
	var lastUsed sql.NullString
	err := s.db.QueryRow(`SELECT id, last_used_at FROM api_keys WHERE token = ?`, token).Scan(&id, &lastUsed)
	if err != nil {
		return false
	}
	// Throttle last_used_at updates to once every 60 seconds to avoid SQLite lock contention
	shouldUpdate := true
	if lastUsed.Valid && lastUsed.String != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", lastUsed.String); err == nil {
			if time.Since(t) < 60*time.Second {
				shouldUpdate = false
			}
		}
	}
	if shouldUpdate {
		now := time.Now().UTC().Format("2006-01-02 15:04:05")
		_, _ = s.db.Exec(`UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now, id)
	}
	return true
}

// GetPrimaryAPIKey returns the most recently used or primary API key token.
// If no keys exist, it automatically provisions an initial "Default Key".
func (s *Store) GetPrimaryAPIKey() string {
	var token string
	err := s.db.QueryRow(`SELECT token FROM api_keys ORDER BY last_used_at DESC, id ASC LIMIT 1`).Scan(&token)
	if err == nil && token != "" {
		return token
	}
	rec, err := s.CreateAPIKey("Default Key")
	if err == nil && rec != nil {
		return rec.Token
	}
	return ""
}

// SyncAPIKeysToFile writes active API keys to ~/.config/m3tal/api_keys.json (or ConfigDir) for easy plugin/daemon discovery.
func (s *Store) SyncAPIKeysToFile() error {
	keys, err := s.ListAPIKeys()
	if err != nil {
		return err
	}
	var targetDir string
	if s.ConfigDir != "" {
		targetDir = s.ConfigDir
	} else {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			targetDir = filepath.Join(home, ".config", "m3tal")
		} else {
			targetDir = "/etc/m3tal"
		}
	}
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return err
	}
	filePath := filepath.Join(targetDir, "api_keys.json")
	payload := map[string]any{
		"primary_token": "",
		"keys":          keys,
	}
	if len(keys) > 0 {
		payload["primary_token"] = keys[0].Token
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(targetDir, "api_keys-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(tmpPath, 0600)
	return os.Rename(tmpPath, filePath)
}
