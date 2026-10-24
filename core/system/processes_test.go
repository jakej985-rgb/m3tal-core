package system

import (
	"testing"
)

func TestGetProcesses(t *testing.T) {
	procs, err := GetProcesses(10)
	if err != nil {
		t.Fatalf("GetProcesses failed: %v", err)
	}
	if len(procs) == 0 {
		t.Log("Warning: 0 processes returned, but no error")
	} else {
		t.Logf("Retrieved %d processes, top process: %s (PID %d)", len(procs), procs[0].Name, procs[0].PID)
		if procs[0].PID <= 0 {
			t.Errorf("Expected positive PID, got %d", procs[0].PID)
		}
	}
}

func TestGetSystemSpecs(t *testing.T) {
	specs, err := GetSystemSpecs()
	if err != nil {
		t.Fatalf("GetSystemSpecs failed: %v", err)
	}
	if specs == nil {
		t.Fatal("Expected non-nil SystemSpecs")
	}
	if specs.Hostname == "" {
		t.Error("Expected non-empty hostname")
	}
	t.Logf("System specs: OS=%s, Platform=%s, Kernel=%s, CPU=%s (%d cores / %d threads)",
		specs.OS, specs.Platform, specs.KernelVersion, specs.CPUModel, specs.CPUCores, specs.CPUThreads)
}

func TestResolveProcessName(t *testing.T) {
	tests := []struct {
		name     string
		procName string
		cmdline  string
		expected string
	}{
		{
			name:     "Un-sandboxed zygote",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=zygote --no-zygote-sandbox --crashpad-handler-pid=123 --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-zygote",
		},
		{
			name:     "Sandboxed zygote (SB)",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=zygote --crashpad-handler-pid=123 --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-zygote (SB)",
		},
		{
			name:     "Renderer process with client ID",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=renderer --renderer-client-id=5 --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-render (#5)",
		},
		{
			name:     "Renderer process without client ID",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=renderer --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-render",
		},
		{
			name:     "GPU process",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=gpu-process --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-gpu",
		},
		{
			name:     "Network utility",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=utility --utility-sub-type=network.mojom.NetworkService --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-net",
		},
		{
			name:     "Storage utility",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --type=utility --utility-sub-type=storage.mojom.StorageService --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-storage",
		},
		{
			name:     "Crashpad handler",
			procName: "chrome_crashpad",
			cmdline:  "/opt/google/chrome/chrome_crashpad_handler --monitor-self --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-crashpad",
		},
		{
			name:     "Main GUI window",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome --app=http://localhost:5050/gui/ --user-data-dir=/home/m3tal/.config/m3tal/browser-profile",
			expected: "m3tal-chrome-gui",
		},
		{
			name:     "M3TAL UI launcher",
			procName: "m3tal",
			cmdline:  "/usr/bin/m3tal ui",
			expected: "m3tal-gui",
		},
		{
			name:     "Unrelated chrome process",
			procName: "chrome",
			cmdline:  "/opt/google/chrome/chrome https://google.com",
			expected: "chrome",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveProcessName(tc.procName, tc.cmdline)
			if got != tc.expected {
				t.Errorf("ResolveProcessName(%q, %q) = %q; want %q", tc.procName, tc.cmdline, got, tc.expected)
			}
		})
	}
}
