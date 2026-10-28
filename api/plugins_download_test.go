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
	path, name := findPluginDeb("m3tal-plugin-mesh-test")
	if path != archive || name != filepath.Base(archive) {
		t.Fatalf("findPluginDeb() = (%q, %q), want (%q, %q)", path, name, archive, filepath.Base(archive))
	}
}

func TestFindPluginDebPrefersVersionedArchive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "apps", "m3tal-plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "m3tal-plugin-mesh-test.deb"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	versioned := filepath.Join(dir, "m3tal-plugin-mesh-test_0.2.0_amd64.deb")
	if err := os.WriteFile(versioned, []byte("versioned"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, name := findPluginDeb("m3tal-plugin-mesh-test")
	if path != versioned || name != filepath.Base(versioned) {
		t.Fatalf("findPluginDeb() = (%q, %q), want versioned archive (%q)", path, name, versioned)
	}
}

func TestFindPluginDebKeepsExactMeshOverClusterAlias(t *testing.T) {
	dir := t.TempDir()
	pluginDebSearchDirs = []string{dir}
	t.Cleanup(func() { pluginDebSearchDirs = nil })

	mesh := filepath.Join(dir, "m3tal-plugin-mesh.deb")
	if err := os.WriteFile(mesh, []byte("mesh"), 0o644); err != nil {
		t.Fatal(err)
	}
	cluster := filepath.Join(dir, "m3tal-plugin-mesh-cluster_0.2.0_amd64.deb")
	if err := os.WriteFile(cluster, []byte("cluster"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, name := findPluginDeb("mesh")
	if path != mesh || name != filepath.Base(mesh) {
		t.Fatalf("findPluginDeb(mesh) = (%q, %q), want exact mesh archive (%q)", path, name, mesh)
	}

	path, name = findPluginDeb("m3tal-plugin-mesh-cluster")
	if path != cluster || name != filepath.Base(cluster) {
		t.Fatalf("findPluginDeb(cluster package) = (%q, %q), want (%q)", path, name, cluster)
	}
}

func TestPluginInstallSearchTargetsPrefersExplicitPackage(t *testing.T) {
	got := pluginInstallSearchTargets("mesh", "m3tal-plugin-mesh", "mesh")
	if len(got) == 0 || got[0] != "m3tal-plugin-mesh" {
		t.Fatalf("pluginInstallSearchTargets() = %v, want explicit package first", got)
	}
}
