package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	sessionDirName  = "gopherkeeper"
	sessionFileName = "session.json"
)

// Ошибки локального хранения online-сессии Клиента.
var (
	// ErrNotFound означает, что online-сессия не найдена.
	ErrNotFound = errors.New("session not found")

	// ErrExpired означает, что срок действия online-сессии истёк.
	ErrExpired = errors.New("session expired")

	// ErrServerMismatch означает, что сохранённая online-сессия относится к другому Серверу.
	ErrServerMismatch = errors.New("session belongs to another server")

	// ErrInvalid означает, что данные online-сессии повреждены или имеют неподдерживаемый формат.
	ErrInvalid = errors.New("invalid session")
)

type nowFunc func() time.Time

// Session содержит данные online-сессии Клиента.
type Session struct {
	// ServerAddress содержит адрес Сервера, для которого сохранена сессия.
	ServerAddress string `json:"server_address"`

	// AccessToken содержит bearer token для авторизованных online-запросов.
	AccessToken string `json:"access_token"`

	// ExpiresAt содержит время истечения срока действия token'а.
	ExpiresAt time.Time `json:"expires_at"`
}

// FileStorage хранит online-сессию Клиента в JSON-файле.
type FileStorage struct {
	path string
	now  nowFunc
}

// NewFileStorage создаёт файловое хранилище online-сессии.
//
// Если path пустой, используется путь по умолчанию внутри os.UserCacheDir():
// gopherkeeper/session.json.
func NewFileStorage(path string) (*FileStorage, error) {
	return newFileStorage(path, time.Now)
}

func newFileStorage(path string, now nowFunc) (*FileStorage, error) {
	if now == nil {
		return nil, errors.New("session clock is required")
	}

	resolvedPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	return &FileStorage{path: resolvedPath, now: now}, nil
}

// Path возвращает фактический путь к session-файлу.
func (s *FileStorage) Path() string {
	return s.path
}

// Save атомарно сохраняет online-сессию в файл.
func (s *FileStorage) Save(session Session) error {
	if err := s.validate(session); err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := ensureSessionDirectory(dir); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	data = append(data, '\n')

	tempFile, err := os.CreateTemp(dir, ".session-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary session file: %w", err)
	}

	tempPath := tempFile.Name()
	cleanup := true

	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set temporary session file permissions: %w", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write temporary session file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary session file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace session file: %w", err)
	}

	cleanup = false

	if err := os.Chmod(s.path, 0o600); err != nil {
		return fmt.Errorf("set session file permissions: %w", err)
	}

	return nil
}

// Delete удаляет сохранённую online-сессию.
//
// Отсутствие session-файла не считается ошибкой: целевое состояние уже достигнуто.
func (s *FileStorage) Delete() error {
	if err := os.Remove(s.path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("delete session file: %w", err)
	}

	return nil
}

// Load читает online-сессию, проверяет срок действия и привязку к Серверу.
func (s *FileStorage) Load(expectedServerAddress string) (Session, error) {
	if expectedServerAddress == "" {
		return Session{}, fmt.Errorf("%w: expected server address is required", ErrInvalid)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, ErrNotFound
		}

		return Session{}, fmt.Errorf("read session file: %w", err)
	}

	var session Session

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&session); err != nil {
		return Session{}, fmt.Errorf("%w: decode JSON", ErrInvalid)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Session{}, fmt.Errorf("%w: extra JSON data", ErrInvalid)
	}

	if err := s.validate(session); err != nil {
		return Session{}, err
	}
	if session.ServerAddress != expectedServerAddress {
		return Session{}, ErrServerMismatch
	}

	return session, nil
}

func resolvePath(path string) (string, error) {
	if path != "" {
		return path, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}

	return filepath.Join(cacheDir, sessionDirName, sessionFileName), nil
}

func (s *FileStorage) validate(session Session) error {
	if session.ServerAddress == "" {
		return fmt.Errorf("%w: server address is required", ErrInvalid)
	}
	if session.AccessToken == "" {
		return fmt.Errorf("%w: access token is required", ErrInvalid)
	}
	if session.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: expiration time is required", ErrInvalid)
	}
	if !session.ExpiresAt.After(s.now()) {
		return ErrExpired
	}

	return nil
}

func ensureSessionDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("session directory path is not a directory: %s", dir)
		}

		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect session directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("set session directory permissions: %w", err)
	}

	return nil
}
