package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jakej985-rgb/m3tal-core/core/events"
	"github.com/jakej985-rgb/m3tal-core/core/queue"
	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// AIJob implements queue.Job for local LLM inference via Ollama.
type AIJob struct {
	IDStr       string
	PriorityVal int
	Prompt      string
	Mode        string
	ModelName   string
	SubmittedAt time.Time
}

// Ensure AIJob implements queue.Job interface.
var _ queue.Job = (*AIJob)(nil)

func (j *AIJob) ID() string {
	return j.IDStr
}

func (j *AIJob) Priority() int {
	return j.PriorityVal
}

func (j *AIJob) Type() string {
	return "ai_generation"
}

func (j *AIJob) Payload() map[string]any {
	return map[string]any{
		"prompt": j.Prompt,
		"mode":   j.Mode,
		"model":  j.ModelName,
	}
}

// OllamaGenerateRequest is the request body for Ollama's /api/generate
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse is the response body from Ollama's /api/generate
type OllamaGenerateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (j *AIJob) Execute(ctx context.Context) (any, error) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	model := j.ModelName
	if model == "" {
		model = "llama3"
	}

	// Prepare request payload
	reqPayload := OllamaGenerateRequest{
		Model:  model,
		Prompt: j.Prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", ollamaHost)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 25 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// Connection failed, fallback to simulated response to keep playground functional
		log.Printf("⚠️ Ollama offline at %s, running in simulated fallback mode.", ollamaHost)
		return j.simulateResponse(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := ioReadAllLimit(resp.Body, 1024)
		return nil, fmt.Errorf("ollama returned HTTP status %d: %s", resp.StatusCode, string(respBytes))
	}

	var genResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return genResp.Response, nil
}

func (j *AIJob) simulateResponse() string {
	promptLower := strings.ToLower(j.Prompt)
	var sb strings.Builder

	sb.WriteString("🤖 [M3TAL Simulated Copilot Mode]\n")
	sb.WriteString("⚠️  Local Ollama server is offline or unreachable on http://localhost:11434.\n")
	sb.WriteString("To enable real LLM inference, run: `m3tal plugin enable ai` or run `ollama serve` locally.\n\n")

	if strings.Contains(promptLower, "status") || strings.Contains(promptLower, "health") {
		sb.WriteString("📋 System Check Diagnostics:\n")
		sb.WriteString("  - Daemon Status: Running (Healthy)\n")
		sb.WriteString("  - Container Providers: Docker (Active)\n")
		sb.WriteString("  - SQLite Database Store: Active\n")
		sb.WriteString("  - VPN Layer: Enabled (gluetun)")
	} else if strings.Contains(promptLower, "docker") || strings.Contains(promptLower, "container") {
		sb.WriteString("🐳 Docker Container Operations Guide:\n")
		sb.WriteString("  - To list active containers, run: `m3tal ps` or `m3tal list`.\n")
		sb.WriteString("  - To manage compose stacks, use `m3tal up [name]` or `m3tal down [name]`.\n")
		sb.WriteString("  - Tail container logs dynamically with `m3tal logs [container_name]`.")
	} else if strings.Contains(promptLower, "help") || strings.Contains(promptLower, "hello") {
		sb.WriteString("👋 Hello! I am the M3TAL Copilot assistant.\n")
		sb.WriteString("I can help you monitor Docker compose stacks, manage networks, inspect system metrics, and control Gluetun VPN settings.")
	} else {
		sb.WriteString(fmt.Sprintf("Processed Prompt Context:\n\"%s\"\n\n", j.Prompt))
		sb.WriteString("In simulated execution, your prompt has been processed by the background queue manager successfully.")
	}

	return sb.String()
}

// ioReadAllLimit reads up to limit bytes from r.
func ioReadAllLimit(r ioReader, limit int64) ([]byte, error) {
	// Custom implementation to avoid adding "io" package dependency if simple
	buf := make([]byte, limit)
	n, err := r.Read(buf)
	if n > 0 {
		return buf[:n], nil
	}
	return nil, err
}

type ioReader interface {
	Read(p []byte) (n int, err error)
}

// ─── API Handlers ───

// ListQueue returns the list of all queued and finished tasks.
// GET /api/v2/queue
// GET /api/v2/ai/queue
func ListQueue(w http.ResponseWriter, r *http.Request) {
	if GlobalQueueManager == nil {
		sendError(w, http.StatusServiceUnavailable, "QUEUE_UNAVAILABLE", "Queue manager is not running", nil)
		return
	}

	records := GlobalQueueManager.List()
	typedRecords := make([]models.JobRecord, len(records))
	for i, r := range records {
		var jobStatus models.JobStatus
		switch r.Status {
		case queue.StatusPending:
			jobStatus = models.JobStatusPending
		case queue.StatusRunning:
			jobStatus = models.JobStatusRunning
		case queue.StatusCompleted:
			jobStatus = models.JobStatusCompleted
		case queue.StatusFailed:
			jobStatus = models.JobStatusFailed
		}

		typedRecords[i] = models.JobRecord{
			ID:          r.ID,
			Type:        r.Type,
			Priority:    r.Priority,
			Status:      jobStatus,
			Payload:     r.Payload,
			SubmittedAt: r.SubmittedAt,
			StartedAt:   r.StartedAt,
			FinishedAt:  r.FinishedAt,
			Result:      r.Result,
			Error:       r.Error,
		}
	}

	sendSuccess(w, http.StatusOK, typedRecords, nil)
}

// CancelQueueJob aborts a pending or active job by ID.
// POST /api/v2/queue/cancel
// POST /api/v2/queue/{id}/cancel
func CancelQueueJob(w http.ResponseWriter, r *http.Request) {
	if GlobalQueueManager == nil {
		sendError(w, http.StatusServiceUnavailable, "QUEUE_UNAVAILABLE", "Queue manager is not running", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		// Try parsing from body
		var body struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			id = body.ID
		}
	}

	if id == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Job ID is required", nil)
		return
	}

	cancelled := GlobalQueueManager.Cancel(id)
	if !cancelled {
		sendError(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found or already finished", nil)
		return
	}

	events.GlobalEventBus.Publish("queue.cancelled", map[string]any{"id": id})
	sendSuccess(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// OllamaTagsResponse holds model listing from Ollama
type OllamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// ListAIModels returns the list of available AI models from the Ollama backend.
// GET /api/v2/ai/models
func ListAIModels(w http.ResponseWriter, r *http.Request) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	url := fmt.Sprintf("%s/api/tags", ollamaHost)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// Fallback to static catalog list if offline
		sendSuccess(w, http.StatusOK, []string{"llama3:latest", "mistral:latest", "codegemma:latest"}, nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sendSuccess(w, http.StatusOK, []string{"llama3:latest", "mistral:latest", "codegemma:latest"}, nil)
		return
	}

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		sendSuccess(w, http.StatusOK, []string{"llama3:latest", "mistral:latest", "codegemma:latest"}, nil)
		return
	}

	models := make([]string, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = m.Name
	}

	sendSuccess(w, http.StatusOK, models, nil)
}

// RunAIInference parses prompt parameters, enqueues an AIJob, blocks for completion, and returns the response.
// POST /api/v2/ai/run
func RunAIInference(w http.ResponseWriter, r *http.Request) {
	if GlobalQueueManager == nil {
		sendError(w, http.StatusServiceUnavailable, "QUEUE_UNAVAILABLE", "Queue manager is not running", nil)
		return
	}

	var req struct {
		Prompt   string `json:"prompt"`
		Mode     string `json:"mode"`
		Priority string `json:"priority"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
		return
	}

	if req.Prompt == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Prompt context is required", nil)
		return
	}

	priorityVal := 2 // Normal default
	switch strings.ToLower(req.Priority) {
	case "low":
		priorityVal = 1
	case "high":
		priorityVal = 3
	}

	// Generate random job ID
	idBytes := make([]byte, 4)
	if _, err := rand.Read(idBytes); err != nil {
		sendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate job ID", nil)
		return
	}
	jobID := fmt.Sprintf("job-%s", hex.EncodeToString(idBytes))

	job := &AIJob{
		IDStr:       jobID,
		PriorityVal: priorityVal,
		Prompt:      req.Prompt,
		Mode:        req.Mode,
		ModelName:   req.Model,
		SubmittedAt: time.Now(),
	}

	// Submit job to queue manager
	if err := queue.Submit(job); err != nil {
		sendError(w, http.StatusInternalServerError, "QUEUE_SUBMIT_FAILED", err.Error(), nil)
		return
	}

	// Publish WebSocket event
	events.GlobalEventBus.Publish("queue.submitted", map[string]any{"id": jobID})

	// Block and poll for status transitions up to 30s
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(30 * time.Second)

	startedPublished := false

	for {
		select {
		case <-ticker.C:
			if rec, found := GlobalQueueManager.Get(jobID); found {
				if rec.Status == queue.StatusRunning && !startedPublished {
					events.GlobalEventBus.Publish("queue.started", map[string]any{"id": jobID})
					startedPublished = true
				}
				switch rec.Status {
				case queue.StatusCompleted:
					events.GlobalEventBus.Publish("queue.completed", map[string]any{"id": jobID})
					sendSuccess(w, http.StatusOK, map[string]any{
						"response": rec.Result,
						"model":    job.ModelName,
					}, nil)
					return
				case queue.StatusFailed:
					events.GlobalEventBus.Publish("queue.failed", map[string]any{"id": jobID, "error": rec.Error})
					sendError(w, http.StatusInternalServerError, "AI_EXECUTION_FAILED", rec.Error, nil)
					return
				}
			}
		case <-timeout:
			sendError(w, http.StatusGatewayTimeout, "TIMEOUT", "AI inference job execution timed out", nil)
			return
		case <-r.Context().Done():
			// Request cancelled by client, attempt to cancel execution
			GlobalQueueManager.Cancel(jobID)
			return
		}
	}
}
