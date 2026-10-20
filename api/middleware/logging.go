package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/jakej985-rgb/m3tal-core/pkg/models"
)

// responseWriterWrapper wraps http.ResponseWriter to capture status code.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// RequestLogger is a middleware that logs incoming requests and processing duration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriterWrapper{ResponseWriter: w}

		defer func() {
			duration := time.Since(start)
			status := wrapper.statusCode
			if status == 0 {
				status = http.StatusOK
			}
			log.Printf("[API] %d %s %s (took %v)", status, r.Method, r.URL.Path, duration)
		}()

		next.ServeHTTP(wrapper, r)
	})
}

// Recoverer maps unhandled panics to a structured INTERNAL_ERROR API response.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				log.Printf("[API] PANIC recovered: %v\n%s", err, string(stack))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				resp := map[string]any{
					"status": "error",
					"data":   nil,
					"meta":   map[string]any{},
					"error": &models.ErrorResponse{
						Code:    "INTERNAL_ERROR",
						Message: fmt.Sprintf("Internal server error: %v", err),
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
