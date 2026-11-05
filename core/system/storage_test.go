package system

import (
	"path/filepath"
	"testing"
)

func TestGetDiskPartitions(t *testing.T) {
	partitions, err := GetDiskPartitions()
	if err != nil {
		t.Fatalf("GetDiskPartitions failed: %v", err)
	}

	for i, p := range partitions {
		if p.Device == "" {
			t.Errorf("Partition %d: expected non-empty Device", i)
		}
		if p.Mountpoint == "" {
			t.Errorf("Partition %d: expected non-empty Mountpoint", i)
		}
		if p.Total <= 0 {
			t.Errorf("Partition %d (%s): expected Total > 0, got %f", i, p.Mountpoint, p.Total)
		}
	}
}

func TestGetDeviceLabel(t *testing.T) {
	// For an unknown/fake device path, it should fallback to filepath.Base
	fakePath := "/dev/fake_test_disk_xyz99"
	expected := filepath.Base(fakePath)
	actual := getDeviceLabel(fakePath)
	if actual != expected {
		t.Errorf("getDeviceLabel(%q) = %q, want %q", fakePath, actual, expected)
	}
}

func BenchmarkGetDiskPartitions(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetDiskPartitions()
		if err != nil {
			b.Fatalf("GetDiskPartitions failed: %v", err)
		}
	}
}
