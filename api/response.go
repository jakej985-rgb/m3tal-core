package api

import (
	"encoding/json"
	"net/http"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// APIResponse represents the standardized JSON structure for all M3TAL API responses.
type APIResponse struct {
	Status string                `json:"status"`
	Data   any                   `json:"data"`           // Explicitly serialized to null if nil
	Meta   any                   `json:"meta,omitempty"` // Omitted if empty/nil
	Error  *models.ErrorResponse `json:"error"`          // Explicitly serialized to null if nil
}

// writeJSONResponse encodes a standardized APIResponse as JSON and writes it to the response.
func writeJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// sendSuccess writes a standard successful response.
func sendSuccess(w http.ResponseWriter, status int, data any, meta any) {
	if meta == nil {
		meta = map[string]any{}
	}
	writeJSONResponse(w, status, APIResponse{
		Status: "success",
		Data:   data,
		Meta:   meta,
		Error:  nil,
	})
}

// sendError writes a standard error response.
func sendError(w http.ResponseWriter, status int, code string, message string, details any) {
	writeJSONResponse(w, status, APIResponse{
		Status: "error",
		Data:   nil,
		Meta:   map[string]any{},
		Error: &models.ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
