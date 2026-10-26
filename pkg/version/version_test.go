package version

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	v := GetVersion()
	if v == "" {
		t.Fatalf("expected non-empty version string")
	}
}

func TestGetInfo(t *testing.T) {
	info := GetInfo()
	if info.Version == "" {
		t.Errorf("expected non-empty version")
	}
	if info.Display != "v"+info.Version {
		t.Errorf("expected display 'v%s', got %q", info.Version, info.Display)
	}
	if info.APIVersion != "v2.0" {
		t.Errorf("expected APIVersion 'v2.0', got %q", info.APIVersion)
	}
	if info.Codename == "" {
		t.Errorf("expected non-empty Codename")
	}
}
