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
