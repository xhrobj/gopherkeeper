package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
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

// UserRegistrar регистрирует нового пользователя.
type UserRegistrar interface {
	Register(ctx context.Context, login, password string) (model.User, error)
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHandler создаёт основной HTTP-handler Сервера.
func NewHandler(database DatabasePinger, registrar UserRegistrar) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(database))
	mux.HandleFunc("POST /api/v1/auth/register", registerHandler(registrar))

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
