package usecase

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// HealthChecker описывает удалённую проверку доступности Сервера.
type HealthChecker interface {
	Health(ctx context.Context) (string, error)
}

// UserGateway описывает удалённые операции с пользователями, необходимые application-слою.
type UserGateway interface {
	Register(ctx context.Context, login, password string) (model.User, error)
	Login(ctx context.Context, login, password string) (model.Authentication, error)
	CurrentUser(ctx context.Context, accessToken string) (model.User, error)
}

// SessionStorage описывает локальное хранилище online-сессии.
type SessionStorage interface {
	Save(stored session.Session) error
	Load() (session.Session, error)
	Delete() error
}

// SessionStorageProvider лениво создаёт локальное хранилище online-сессии.
type SessionStorageProvider func() (SessionStorage, error)

// Application выполняет клиентские online- и offline-сценарии поверх удалённых
// gateway, provider'ов локальной online-сессии и зашифрованного кеша.
type Application struct {
	health        HealthChecker
	users         UserGateway
	records       RecordGateway
	sessions      SessionStorageProvider
	syncCaches    SyncCacheRepositoryProvider
	offlineCaches OfflineCacheRepositoryProvider
}

// OfflineApplication выполняет только offline read-only сценарии поверх
// существующего зашифрованного локального кеша.
type OfflineApplication struct {
	offlineCaches OfflineCacheRepositoryProvider
}

// New создаёт application-приложение из готовых зависимостей.
func New(
	health HealthChecker,
	users UserGateway,
	records RecordGateway,
	sessions SessionStorageProvider,
	syncCaches SyncCacheRepositoryProvider,
	offlineCaches OfflineCacheRepositoryProvider,
) *Application {
	return &Application{
		health:        health,
		users:         users,
		records:       records,
		sessions:      sessions,
		syncCaches:    syncCaches,
		offlineCaches: offlineCaches,
	}
}

// NewOffline создаёт application-приложение только для чтения существующего
// зашифрованного локального кеша без сетевых и session-зависимостей.
func NewOffline(
	offlineCaches OfflineCacheRepositoryProvider,
) *OfflineApplication {
	return &OfflineApplication{
		offlineCaches: offlineCaches,
	}
}
