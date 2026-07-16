package usecase

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Application выполняет клиентские online- и offline-сценарии поверх удалённых
// gateway, provider'ов локальной online-сессии и зашифрованного кеша.
type Application struct {
	users         UserGateway
	records       RecordGateway
	sessions      SessionStorageProvider
	syncCaches    SyncCacheRepositoryProvider
	offlineCaches OfflineCacheRepositoryProvider
	serverAddress string
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
	Load(expectedServerAddress string) (session.Session, error)
}

// SessionStorageProvider лениво создаёт локальное хранилище online-сессии.
type SessionStorageProvider func() (SessionStorage, error)

// New создаёт application-приложение из готовых зависимостей.
func New(
	users UserGateway,
	records RecordGateway,
	sessions SessionStorageProvider,
	syncCaches SyncCacheRepositoryProvider,
	offlineCaches OfflineCacheRepositoryProvider,
	serverAddress string,
) *Application {
	return &Application{
		users:         users,
		records:       records,
		sessions:      sessions,
		syncCaches:    syncCaches,
		offlineCaches: offlineCaches,
		serverAddress: serverAddress,
	}
}

// NewOffline создаёт application-приложение только для чтения существующего
// зашифрованного локального кеша без сетевых и session-зависимостей.
func NewOffline(
	offlineCaches OfflineCacheRepositoryProvider,
	serverAddress string,
) *Application {
	return &Application{
		offlineCaches: offlineCaches,
		serverAddress: serverAddress,
	}
}
