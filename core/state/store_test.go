package state

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%s): %v", path, err)
	}
	db.ConfigDir = dir
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenAndMigrate(t *testing.T) {
	db := tempDB(t)
	if db == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestGetStatePath(t *testing.T) {
	// With env override
	os.Setenv("M3TAL_STATE_DB", "/tmp/test-state.db")
	defer os.Unsetenv("M3TAL_STATE_DB")

	p := GetStatePath()
	if p != "/tmp/test-state.db" {
		t.Errorf("expected /tmp/test-state.db, got %s", p)
	}
}

func TestAPIKeysCRUD(t *testing.T) {
	db := tempDB(t)

	// List initial provisioned key
	keys, err := db.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 initial key, got %d", len(keys))
	}
	if keys[0].Name != "Default Key" || len(keys[0].Token) != 32 {
		t.Fatalf("unexpected initial key: %+v", keys[0])
	}

	// Create new key for mesh
	newKey, err := db.CreateAPIKey("Mesh Fleet", nil, DirectionLocal, "", "")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if newKey.Name != "Mesh Fleet" {
		t.Errorf("expected name 'Mesh Fleet', got %s", newKey.Name)
	}
	if len(newKey.Token) != 32 {
		t.Errorf("expected 32 hex chars token, got len=%d", len(newKey.Token))
	}

	// Empty name rejected
	if _, err := db.CreateAPIKey("   ", nil, DirectionLocal, "", ""); err == nil {
		t.Error("expected error creating API key with empty name")
	}

	// Validation
	if !db.ValidateAPIKey(newKey.Token) {
		t.Errorf("expected ValidateAPIKey(%s) to be true", newKey.Token)
	}
	if db.ValidateAPIKey("non-existent-token") {
		t.Error("expected ValidateAPIKey for non-existent token to be false")
	}

	// Verify last_used_at updated
	keys, err = db.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	found := false
	for _, k := range keys {
		if k.ID == newKey.ID && k.LastUsedAt != nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected last_used_at to be populated after validation")
	}

	// Delete key
	if err := db.DeleteAPIKey(newKey.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}
	if db.ValidateAPIKey(newKey.Token) {
		t.Error("deleted key should not validate")
	}

	// Delete non-existent key returns error
	if err := db.DeleteAPIKey(99999); err == nil {
		t.Error("expected error deleting non-existent API key ID")
	}
}

func TestRecordAPIKeyUsage(t *testing.T) {
	db := tempDB(t)

	rec, err := db.CreateAPIKey("Usage Key", nil, DirectionLocal, "", "")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	// Lookup + record usage without a second SELECT (the Authenticate path).
	looked, ok := db.LookupAPIKey(rec.Token)
	if !ok {
		t.Fatal("LookupAPIKey failed")
	}
	db.RecordAPIKeyUsage(looked)
	if looked.LastUsedAt == nil {
		t.Fatal("expected LastUsedAt to be stamped on the record")
	}

	// A fresh record object for the same key should now throttle the write.
	again, ok := db.LookupAPIKey(rec.Token)
	if !ok {
		t.Fatal("second LookupAPIKey failed")
	}
	before := *again.LastUsedAt
	db.RecordAPIKeyUsage(again)
	keys, err := db.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	for _, k := range keys {
		if k.ID == rec.ID && k.LastUsedAt != nil && *k.LastUsedAt != before {
			t.Errorf("expected throttled usage write to keep %s, got %s", before, *k.LastUsedAt)
		}
	}
}

func TestGetPrimaryAPIKey(t *testing.T) {
	db := tempDB(t)

	// Primary key should be initial default key
	pk := db.GetPrimaryAPIKey()
	if len(pk) != 32 {
		t.Fatalf("expected 32-char primary token, got %q", pk)
	}

	// Create a new key and validate it to update last_used_at
	meshKey, err := db.CreateAPIKey("Mesh Fleet", nil, DirectionLocal, "", "")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if !db.ValidateAPIKey(meshKey.Token) {
		t.Fatal("ValidateAPIKey failed")
	}

	// GetPrimaryAPIKey should now return meshKey
	newPk := db.GetPrimaryAPIKey()
	if newPk != meshKey.Token {
		t.Errorf("expected primary key %s, got %s", meshKey.Token, newPk)
	}

	// Delete all keys
	keys, err := db.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	for _, k := range keys {
		if err := db.DeleteAPIKey(k.ID); err != nil {
			t.Fatalf("DeleteAPIKey(%d): %v", k.ID, err)
		}
	}

	// When empty, GetPrimaryAPIKey auto-provisions a new Default Key
	autoPk := db.GetPrimaryAPIKey()
	if len(autoPk) != 32 {
		t.Fatalf("expected auto-provisioned primary key token, got %q", autoPk)
	}
}
