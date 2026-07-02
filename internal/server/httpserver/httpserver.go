package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const (
	healthCheckTimeout      = 2 * time.Second
	healthStatusOK          = "ok"
	healthStatusUnavailable = "unavailable"
)

// DatabasePinger проверяет доступность PostgreSQL.
type DatabasePinger interface {
	Ping(context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHandler создаёт основной HTTP-handler Сервера.
func NewHandler(database DatabasePinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(database))

	return mux
}

func healthHandler(database DatabasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		status := healthStatusOK

		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		if err := database.Ping(ctx); err != nil {
			statusCode = http.StatusServiceUnavailable
			status = healthStatusUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(healthResponse{
			Status: status,
		}); err != nil {
			return
		}
	}
}
