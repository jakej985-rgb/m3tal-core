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
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenAndMigrate(t *testing.T) {
	db := tempDB(t)
	if db == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestRoutesCRUD(t *testing.T) {
	db := tempDB(t)

	// Create
	id, err := db.CreateRoute("app", "app.local", 8080, "web", "media", false, "")
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id=1, got %d", id)
	}

	// List
	routes, err := db.ListRoutes()
	if err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Service != "app" || routes[0].Domain != "app.local" || routes[0].Port != 8080 {
		t.Errorf("unexpected route: %+v", routes[0])
	}

	// Get by domain
	r, err := db.GetRouteByDomain("app.local")
	if err != nil {
		t.Fatalf("GetRouteByDomain: %v", err)
	}
	if r == nil || r.Service != "app" {
		t.Errorf("unexpected route by domain: %+v", r)
	}

	// Not found
	r, err = db.GetRouteByDomain("nonexistent.local")
	if err != nil {
		t.Fatalf("GetRouteByDomain: %v", err)
	}
	if r != nil {
		t.Error("expected nil for nonexistent domain")
	}

	// Delete
	if err := db.DeleteRoute(id); err != nil {
		t.Fatalf("DeleteRoute: %v", err)
	}
	routes, _ = db.ListRoutes()
	if len(routes) != 0 {
		t.Errorf("expected 0 routes after delete, got %d", len(routes))
	}

	// Delete nonexistent
	if err := db.DeleteRoute(999); err == nil {
		t.Error("expected error deleting nonexistent route")
	}
}

func TestDuplicateDomain(t *testing.T) {
	db := tempDB(t)

	_, err := db.CreateRoute("app", "app.local", 8080, "", "", false, "")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = db.CreateRoute("other", "app.local", 9090, "", "", false, "")
	if err == nil {
		t.Error("expected error for duplicate domain")
	}
}

func TestStacksCRUD(t *testing.T) {
	db := tempDB(t)

	if err := db.UpsertStack("media", "/docker/media-compose.yml"); err != nil {
		t.Fatalf("UpsertStack: %v", err)
	}

	stacks, err := db.ListStacks()
	if err != nil {
		t.Fatalf("ListStacks: %v", err)
	}
	if len(stacks) != 1 || stacks[0].Name != "media" {
		t.Errorf("unexpected stacks: %+v", stacks)
	}

	// Update status
	if err := db.UpdateStackStatus("media", "running"); err != nil {
		t.Fatalf("UpdateStackStatus: %v", err)
	}

	stacks, _ = db.ListStacks()
	if stacks[0].Status != "running" {
		t.Errorf("expected status 'running', got %q", stacks[0].Status)
	}

	// Upsert updates path
	if err := db.UpsertStack("media", "/new/path.yml"); err != nil {
		t.Fatalf("UpsertStack update: %v", err)
	}
	stacks, _ = db.ListStacks()
	if stacks[0].ComposePath != "/new/path.yml" {
		t.Errorf("expected updated path, got %q", stacks[0].ComposePath)
	}
}

func TestMiddlewareCRUD(t *testing.T) {
	db := tempDB(t)

	config := map[string]string{"users": "admin:pass"}
	id, err := db.CreateMiddleware("auth", "basicauth", config)
	if err != nil {
		t.Fatalf("CreateMiddleware: %v", err)
	}

	mws, err := db.ListMiddleware()
	if err != nil {
		t.Fatalf("ListMiddleware: %v", err)
	}
	if len(mws) != 1 || mws[0].Name != "auth" || mws[0].Type != "basicauth" {
		t.Errorf("unexpected middleware: %+v", mws)
	}
	if mws[0].Config["users"] != "admin:pass" {
		t.Errorf("unexpected config: %v", mws[0].Config)
	}

	if err := db.DeleteMiddleware(id); err != nil {
		t.Fatalf("DeleteMiddleware: %v", err)
	}
	mws, _ = db.ListMiddleware()
	if len(mws) != 0 {
		t.Errorf("expected 0 middleware after delete, got %d", len(mws))
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
