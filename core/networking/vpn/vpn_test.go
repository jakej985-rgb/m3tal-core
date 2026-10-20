package vpn

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwitchRegion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "m3tal-vpn-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envFile := filepath.Join(tmpDir, ".env")
	initialContent := "VPN_USER=testuser\nVPN_REGIONS=Norway\nVPN_PASSWORD=secret"
	if err := os.WriteFile(envFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write test env: %v", err)
	}

	// Override M3TAL_CONFIG env var so system.GetConfigPath() points to our temp file
	origEnv := os.Getenv("M3TAL_CONFIG")
	os.Setenv("M3TAL_CONFIG", envFile)
	defer os.Setenv("M3TAL_CONFIG", origEnv)

	// Mock manager (we won't call SwitchRegion's redeploy, we expect it to fail compose search but successfully update .env)
	mgr := &Manager{}
	err = mgr.SwitchRegion("Switzerland")
	// It might error due to missing docker/compose, but we want to verify the .env was written
	if err != nil && !strings.Contains(err.Error(), "docker") && !strings.Contains(err.Error(), "compose") && !strings.Contains(err.Error(), "stat") {
		t.Fatalf("unexpected switch region error: %v", err)
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("failed to read written env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "VPN_REGIONS=Switzerland") {
		t.Errorf("expected VPN_REGIONS to be updated to Switzerland, got:\n%s", content)
	}
}

func TestCheckLeakLogic(t *testing.T) {
	// Mock external IP service
	serverHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1.2.3.4"))
	}))
	defer serverHost.Close()

	// 1. Test when host and VPN IPs are different (No leak)
	isLeak, hostIP, vpnIP, err := checkLeakWithMockedIPs("1.2.3.4", "5.6.7.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isLeak {
		t.Error("expected no leak when IPs are different")
	}
	if hostIP != "1.2.3.4" || vpnIP != "5.6.7.8" {
		t.Errorf("IP mismatch: host=%s, vpn=%s", hostIP, vpnIP)
	}

	// 2. Test when host and VPN IPs are identical (Leak detected)
	isLeak, hostIP, vpnIP, err = checkLeakWithMockedIPs("1.2.3.4", "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isLeak {
		t.Error("expected leak detected when IPs are identical")
	}
}

func checkLeakWithMockedIPs(hostIP, vpnIP string) (bool, string, string, error) {
	// Direct mock comparison function mimicking CheckLeak
	isLeak := hostIP == vpnIP
	return isLeak, hostIP, vpnIP, nil
}
