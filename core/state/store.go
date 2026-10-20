package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for M3TAL state persistence.
type Store struct {
	db *sql.DB
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
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	// Schema migrations for v2 proxy (SSL and Middlewares support)
	_, _ = s.db.Exec("ALTER TABLE routes ADD COLUMN ssl INTEGER DEFAULT 0")
	_, _ = s.db.Exec("ALTER TABLE routes ADD COLUMN middlewares TEXT DEFAULT ''")

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
