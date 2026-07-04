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

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	sessionDirName  = "gkeep"
	sessionFileName = "session.json"
	tokenTypeBearer = "Bearer"
)

// Ошибки локального хранения online-сессии Клиента.
var (
	ErrNotFound       = errors.New("session not found")
	ErrExpired        = errors.New("session expired")
	ErrServerMismatch = errors.New("session belongs to another server")
	ErrInvalid        = errors.New("invalid session")
)

// Clock возвращает текущее время и позволяет фиксировать его в тестах.
type Clock interface {
	Now() time.Time
}

// Session содержит данные online-сессии Клиента.
type Session struct {
	ServerAddress string     `json:"server_address"`
	AccessToken   string     `json:"access_token"`
	TokenType     string     `json:"token_type"`
	ExpiresAt     time.Time  `json:"expires_at"`
	User          model.User `json:"user"`
}

// FileStorage хранит online-сессию Клиента в JSON-файле.
type FileStorage struct {
	path  string
	clock Clock
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

// NewFileStorage создаёт файловое хранилище online-сессии.
//
// Если path пустой, используется путь по умолчанию внутри os.UserCacheDir():
// gkeep/session.json.
func NewFileStorage(path string) (*FileStorage, error) {
	return newFileStorage(path, realClock{})
}

func newFileStorage(path string, clock Clock) (*FileStorage, error) {
	if clock == nil {
		return nil, errors.New("session clock is required")
	}

	resolvedPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	return &FileStorage{path: resolvedPath, clock: clock}, nil
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("set session directory permissions: %w", err)
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

func (s *FileStorage) validate(session Session) error {
	if session.ServerAddress == "" {
		return fmt.Errorf("%w: server address is required", ErrInvalid)
	}
	if session.AccessToken == "" {
		return fmt.Errorf("%w: access token is required", ErrInvalid)
	}
	if session.TokenType != tokenTypeBearer {
		return fmt.Errorf("%w: unsupported token type", ErrInvalid)
	}
	if session.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: expiration time is required", ErrInvalid)
	}
	if !session.ExpiresAt.After(s.clock.Now()) {
		return ErrExpired
	}
	if session.User.ID <= 0 {
		return fmt.Errorf("%w: user id is required", ErrInvalid)
	}
	if session.User.Login == "" {
		return fmt.Errorf("%w: user login is required", ErrInvalid)
	}
	if session.User.CreatedAt.IsZero() {
		return fmt.Errorf("%w: user created_at is required", ErrInvalid)
	}

	return nil
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
