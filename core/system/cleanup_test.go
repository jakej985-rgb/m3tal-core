package system

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{-100, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.input, got, tt.expected)
		}
	}
}

func TestCleanSystem(t *testing.T) {
	// CleanSystem should execute safely without crashing or returning nil
	res, err := CleanSystem()
	if err != nil {
		t.Fatalf("CleanSystem failed: %v", err)
	}
	if res == nil {
		t.Fatal("CleanSystem returned nil result")
	}
	if res.FreedBytes < 0 {
		t.Errorf("FreedBytes should not be negative: %d", res.FreedBytes)
	}
}
