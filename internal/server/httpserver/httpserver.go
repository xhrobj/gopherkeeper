package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
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

// UserRegisterer регистрирует нового пользователя.
type UserRegisterer interface {
	// Register регистрирует нового пользователя.
	Register(ctx context.Context, login, password string) (model.User, error)
}

// UserAuthenticator аутентифицирует пользователя и выпускает bearer token.
type UserAuthenticator interface {
	// Authenticate проверяет учётные данные пользователя.
	Authenticate(ctx context.Context, login, password string) (service.AuthenticationResult, error)
}

// CurrentUserReader возвращает публичные данные текущего пользователя.
type CurrentUserReader interface {
	// FindByID возвращает публичные данные пользователя по идентификатору.
	FindByID(ctx context.Context, id int64) (model.User, error)
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHandler создаёт основной HTTP-handler Сервера.
func NewHandler(
	database DatabasePinger,
	registerer UserRegisterer,
	authenticator UserAuthenticator,
	tokenValidator TokenValidator,
	currentUserReader CurrentUserReader,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(database))
	mux.HandleFunc("POST /api/v1/auth/register", registerHandler(registerer))
	mux.HandleFunc("POST /api/v1/auth/login", loginHandler(authenticator))
	mux.Handle(
		"GET /api/v1/users/me",
		WithAuthentication(currentUserHandler(currentUserReader), tokenValidator),
	)

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
