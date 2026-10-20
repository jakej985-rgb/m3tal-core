package plugins

import (
	"encoding/json"
	"testing"
)

func TestPluginManifestSerialization(t *testing.T) {
	manifest := &PluginManifest{
		APIVersion: APIVersionV1Alpha1,
		Kind:       KindPlugin,
		Metadata: PluginMeta{
			Name:        "docker-monitor",
			Version:     "1.0.0",
			Description: "Docker container telemetry",
			Author:      "m3tal",
		},
		Spec: PluginSpec{
			Command:      "./bin/docker-plugin",
			Capabilities: []Capability{CapabilityCollector, CapabilityTUIWidget, CapabilityActions},
			PollInterval: "2s",
		},
		Config: map[string]any{
			"socket": "/var/run/docker.sock",
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	var parsed PluginManifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	if parsed.Metadata.Name != "docker-monitor" {
		t.Errorf("Expected name 'docker-monitor', got %q", parsed.Metadata.Name)
	}
	if !parsed.HasCapability(CapabilityCollector) {
		t.Errorf("Expected capability 'collector'")
	}
	if !parsed.HasCapability(CapabilityTUIWidget) {
		t.Errorf("Expected capability 'tui_widget'")
	}
	if parsed.HasCapability(CapabilityExporter) {
		t.Errorf("Did not expect capability 'exporter'")
	}
}

func TestRPCRequestResponse(t *testing.T) {
	initParams := InitParams{
		CoreVersion: "2.0.0",
		Config: map[string]any{
			"poll_interval": "1s",
		},
	}
	paramBytes, _ := json.Marshal(initParams)

	req := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  MethodInit,
		Params:  paramBytes,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal RPCRequest: %v", err)
	}

	var parsedReq RPCRequest
	if err := json.Unmarshal(reqBytes, &parsedReq); err != nil {
		t.Fatalf("Failed to unmarshal RPCRequest: %v", err)
	}
	if parsedReq.Method != MethodInit {
		t.Errorf("Expected method %s, got %s", MethodInit, parsedReq.Method)
	}

	// Test Response
	initResult := InitResult{
		Name:         "docker-monitor",
		Version:      "1.0.0",
		Capabilities: []Capability{CapabilityCollector},
		Tabs: []PluginTabSpec{
			{
				ID:       "docker_containers",
				Title:    "Containers",
				Shortcut: "C",
			},
		},
	}
	resBytes, _ := json.Marshal(initResult)

	resp := RPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  resBytes,
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal RPCResponse: %v", err)
	}

	var parsedResp RPCResponse
	if err := json.Unmarshal(respData, &parsedResp); err != nil {
		t.Fatalf("Failed to unmarshal RPCResponse: %v", err)
	}
	if parsedResp.ID != 1 {
		t.Errorf("Expected response ID 1, got %d", parsedResp.ID)
	}
}

func TestCollectResult(t *testing.T) {
	result := CollectResult{
		Status: "ok",
		Metrics: []Metric{
			{
				Name:     "active_containers",
				Value:    5,
				Unit:     "count",
				Category: "docker",
				Status:   "ok",
			},
		},
		Tables: []TableData{
			{
				ID:      "containers_table",
				Title:   "Running Containers",
				Headers: []string{"ID", "Name", "Status"},
				Rows: [][]string{
					{"c1", "web", "running"},
					{"c2", "db", "running"},
				},
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CollectResult: %v", err)
	}

	var parsed CollectResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal CollectResult: %v", err)
	}
	if len(parsed.Metrics) != 1 || parsed.Metrics[0].Value != 5 {
		t.Errorf("Metric value mismatch: %+v", parsed.Metrics)
	}
	if len(parsed.Tables) != 1 || len(parsed.Tables[0].Rows) != 2 {
		t.Errorf("Table data mismatch: %+v", parsed.Tables)
	}
}
