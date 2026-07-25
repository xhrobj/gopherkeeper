package grpcserver

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

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

// TokenValidator проверяет bearer token и возвращает идентификатор пользователя.
type TokenValidator interface {
	// Validate проверяет bearer token.
	Validate(ctx context.Context, token string) (int64, error)
}

// CurrentUserReader возвращает публичные данные текущего пользователя.
type CurrentUserReader interface {
	// FindByID возвращает публичные данные пользователя по идентификатору.
	FindByID(ctx context.Context, id int64) (model.User, error)
}

// RecordManager выполняет серверные сценарии приватных записей.
type RecordManager interface {
	// Create создаёт приватную запись пользователя.
	Create(ctx context.Context, request service.CreateRecordRequest) (model.Record, error)

	// List возвращает открытые поля записей пользователя.
	List(ctx context.Context, userID int64) ([]model.RecordMetadata, error)

	// Get возвращает запись пользователя с расшифрованным payload.
	Get(ctx context.Context, userID int64, recordID string) (model.Record, error)

	// Update изменяет приватную запись пользователя.
	Update(ctx context.Context, request service.UpdateRecordRequest) (model.Record, error)

	// Delete удаляет приватную запись пользователя.
	Delete(ctx context.Context, request service.DeleteRecordRequest) error
}

// Dependencies содержит application-зависимости gRPC transport'а.
type Dependencies struct {
	// Registerer регистрирует новых пользователей.
	Registerer UserRegisterer

	// Authenticator выполняет вход пользователя.
	Authenticator UserAuthenticator

	// TokenValidator проверяет token доступа.
	TokenValidator TokenValidator

	// CurrentUserReader читает данные текущего пользователя.
	CurrentUserReader CurrentUserReader

	// Records выполняет сценарии приватных записей.
	Records RecordManager
}
