package state

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database for M3TAL Core state persistence.
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

// Open creates or opens a SQLite database at the given path.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create database directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("cannot open database %s: %w", path, err)
	}

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("cannot apply pragma %q: %w", pragma, err)
		}
	}

	s := &Store{db: db}
	if err := s.Migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error { return s.db.Close() }

// Ping verifies the database connection is alive.
func (s *Store) Ping() error { return s.db.Ping() }

// Migrate creates the required tables if they don't exist.
func (s *Store) Migrate() error {
	migrations := []string{
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

	var keyCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM api_keys").Scan(&keyCount); err == nil && keyCount == 0 {
		token, err := GenerateRandomAPIKeyToken()
		if err == nil {
			now := time.Now().UTC().Format(time.RFC3339)
			_, _ = s.db.Exec(`INSERT INTO api_keys (name, token, created_at) VALUES (?, ?, ?)`, "Default Key", token, now)
		}
	}
	return nil
}

// GetStatePath returns the default database path, respecting overrides.
func GetStatePath() string {
	if p := os.Getenv("M3TAL_STATE_DB"); p != "" {
		return p
	}
	sysPath := "/var/lib/m3tal/state.db"
	if info, err := os.Stat(filepath.Dir(sysPath)); err == nil && info.IsDir() {
		return sysPath
	}
	return "./data/state.db"
}

// --- API Keys ---

// ListAPIKeys returns all stored API keys.
func (s *Store) ListAPIKeys() ([]APIKeyRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, token, created_at, last_used_at FROM api_keys ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("cannot list API keys: %w", err)
	}
	defer rows.Close()

	keys := []APIKeyRecord{}
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
	return keys, rows.Err()
}

// GenerateRandomAPIKeyToken generates a 32 hex character token.
func GenerateRandomAPIKeyToken() (string, error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(tokenBytes), nil
}

// CreateAPIKey generates a server-side token and stores it with the given name.
func (s *Store) CreateAPIKey(name string) (*APIKeyRecord, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("API key name cannot be empty")
	}

	token, err := GenerateRandomAPIKeyToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := s.db.Exec(`INSERT INTO api_keys (name, token, created_at) VALUES (?, ?, ?)`, trimmedName, token, now)
	if err != nil {
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	rec := &APIKeyRecord{ID: id, Name: trimmedName, Token: token, CreatedAt: now}
	_ = s.SyncAPIKeysToFile()
	return rec, nil
}

// ErrAPIKeyNotFound is returned when operating on an API key that does not exist.
var ErrAPIKeyNotFound = errors.New("API key not found")

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
		return ErrAPIKeyNotFound
	}
	_ = s.SyncAPIKeysToFile()
	return nil
}

// ValidateAPIKey checks a token and records its last usage.
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
	shouldUpdate := true
	if lastUsed.Valid && lastUsed.String != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", lastUsed.String); err == nil && time.Since(t) < 60*time.Second {
			shouldUpdate = false
		}
	}
	if shouldUpdate {
		now := time.Now().UTC().Format("2006-01-02 15:04:05")
		_, _ = s.db.Exec(`UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now, id)
	}
	return true
}

// GetPrimaryAPIKey returns the most recently used API key token.
func (s *Store) GetPrimaryAPIKey() string {
	var token string
	if err := s.db.QueryRow(`SELECT token FROM api_keys ORDER BY last_used_at DESC, id ASC LIMIT 1`).Scan(&token); err == nil && token != "" {
		return token
	}
	if rec, err := s.CreateAPIKey("Default Key"); err == nil && rec != nil {
		return rec.Token
	}
	return ""
}

// SyncAPIKeysToFile writes active API keys for plugin/daemon discovery.
func (s *Store) SyncAPIKeysToFile() error {
	keys, err := s.ListAPIKeys()
	if err != nil {
		return err
	}
	targetDir := s.ConfigDir
	if targetDir == "" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			targetDir = filepath.Join(home, ".config", "m3tal")
		} else {
			targetDir = "/etc/m3tal"
		}
	}
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return err
	}

	primary := s.GetPrimaryAPIKey()
	if primary == "" && len(keys) > 0 {
		primary = keys[0].Token
	}
	data, err := json.MarshalIndent(map[string]any{"primary_token": primary, "keys": keys}, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(targetDir, "api_keys.json")
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
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
