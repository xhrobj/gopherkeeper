package usecase

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Application выполняет клиентские online-сценарии поверх удалённых gateway,
// provider'ов локальной online-сессии и зашифрованного кеша.
type Application struct {
	users         UserGateway
	records       RecordGateway
	sessions      SessionStorageProvider
	caches        SyncCacheRepositoryProvider
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
	caches SyncCacheRepositoryProvider,
	serverAddress string,
) *Application {
	return &Application{
		users:         users,
		records:       records,
		sessions:      sessions,
		caches:        caches,
		serverAddress: serverAddress,
	}
}
