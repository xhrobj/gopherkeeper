package service

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// UserRepository описывает хранение зарегистрированных пользователей.
type UserRepository interface {
	// Create сохраняет пользователя с подготовленным хэшем пароля.
	Create(ctx context.Context, login string, passwordHash []byte) (model.User, error)
}

// PasswordHasher описывает создание хэша пароля.
type PasswordHasher interface {
	// Hash возвращает хэш переданного пароля.
	Hash(password string) ([]byte, error)
}

// RegistrationService реализует сценарий регистрации пользователя.
type RegistrationService struct {
	users     UserRepository
	passwords PasswordHasher
}

// NewRegistrationService создаёт сервис регистрации пользователя.
func NewRegistrationService(
	users UserRepository,
	passwords PasswordHasher,
) *RegistrationService {
	return &RegistrationService{
		users:     users,
		passwords: passwords,
	}
}

// Register валидирует регистрационные данные, хеширует пароль и создаёт пользователя.
func (s *RegistrationService) Register(
	ctx context.Context,
	login string,
	password string,
) (model.User, error) {
	canonicalLogin, err := validateRegistrationCredentials(login, password)
	if err != nil {
		return model.User{}, err
	}

	passwordHash, err := s.passwords.Hash(password)
	if err != nil {
		return model.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, canonicalLogin, passwordHash)
	if err != nil {
		return model.User{}, fmt.Errorf("register user: %w", err)
	}

	return user, nil
}
