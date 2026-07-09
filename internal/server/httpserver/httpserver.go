package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const (
	healthCheckTimeout      = 2 * time.Second
	healthStatusOK          = "ok"
	healthStatusUnavailable = "unavailable"
)

// DatabasePinger проверяет доступность PostgreSQL.
type DatabasePinger interface {
	// Ping проверяет доступность базы данных.
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

// Dependencies содержит зависимости HTTP-обработчиков Сервера.
type Dependencies struct {
	// Database проверяет доступность PostgreSQL.
	Database DatabasePinger

	// Registerer регистрирует новых пользователей.
	Registerer UserRegisterer

	// Authenticator выполняет вход пользователя.
	Authenticator UserAuthenticator

	// TokenValidator проверяет токен доступа.
	TokenValidator middleware.TokenValidator

	// CurrentUserReader читает данные текущего пользователя.
	CurrentUserReader CurrentUserReader

	// Records выполняет сценарии приватных записей.
	Records RecordManager
}

// NewHandler создаёт основной HTTP-handler Сервера.
func NewHandler(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler(deps.Database))
	mux.HandleFunc("POST /api/v1/auth/register", registerHandler(deps.Registerer))
	mux.HandleFunc("POST /api/v1/auth/login", loginHandler(deps.Authenticator))
	mux.Handle(
		"GET /api/v1/users/me",
		middleware.WithAuthentication(currentUserHandler(deps.CurrentUserReader), deps.TokenValidator),
	)
	mux.Handle(
		"POST /api/v1/records",
		middleware.WithAuthentication(createRecordHandler(deps.Records), deps.TokenValidator),
	)
	mux.Handle(
		"GET /api/v1/records",
		middleware.WithAuthentication(listRecordsHandler(deps.Records), deps.TokenValidator),
	)
	mux.Handle(
		"GET /api/v1/records/{id}",
		middleware.WithAuthentication(getRecordHandler(deps.Records), deps.TokenValidator),
	)
	mux.Handle(
		"PUT /api/v1/records/{id}",
		middleware.WithAuthentication(updateRecordHandler(deps.Records), deps.TokenValidator),
	)
	mux.Handle(
		"DELETE /api/v1/records/{id}",
		middleware.WithAuthentication(deleteRecordHandler(deps.Records), deps.TokenValidator),
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
