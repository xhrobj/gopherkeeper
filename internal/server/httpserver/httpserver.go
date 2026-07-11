package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const (
	healthCheckTimeout      = 2 * time.Second
	healthStatusOK          = "ok"
	healthStatusUnavailable = "unavailable"
	recordByIDPath          = "/records/{id}"
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
	router := chi.NewRouter()

	router.Method(http.MethodGet, "/health", healthHandler(deps.Database))
	router.Route("/api/v1", func(router chi.Router) {
		router.Route("/auth", func(router chi.Router) {
			router.Method(http.MethodPost, "/register", registerHandler(deps.Registerer))
			router.Method(http.MethodPost, "/login", loginHandler(deps.Authenticator))
		})

		router.Group(func(router chi.Router) {
			router.Use(func(handler http.Handler) http.Handler {
				return middleware.WithAuthentication(handler, deps.TokenValidator)
			})

			router.Method(http.MethodGet, "/users/me", currentUserHandler(deps.CurrentUserReader))
			router.Method(http.MethodPost, "/records", createRecordHandler(deps.Records))
			router.Method(http.MethodGet, "/records", listRecordsHandler(deps.Records))
			router.Method(http.MethodGet, recordByIDPath, getRecordHandler(deps.Records))
			router.Method(http.MethodPut, recordByIDPath, updateRecordHandler(deps.Records))
			router.Method(http.MethodDelete, recordByIDPath, deleteRecordHandler(deps.Records))
		})
	})

	return router
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
