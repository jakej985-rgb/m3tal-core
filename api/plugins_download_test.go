package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindPluginDebMatchesExactPackageArchive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "apps", "m3tal-plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "m3tal-plugin-mesh-test.deb")
	if err := os.WriteFile(archive, []byte("deb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "m3tal-plugin-mesh-test_0.1.0_amd64.deb"), []byte("older build"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, name := findPluginDeb("m3tal-plugin-mesh-test")
	if path != archive || name != filepath.Base(archive) {
		t.Fatalf("findPluginDeb() = (%q, %q), want (%q, %q)", path, name, archive, filepath.Base(archive))
	}
}
