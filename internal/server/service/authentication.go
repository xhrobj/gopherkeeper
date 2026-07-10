package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// ErrInvalidCredentials означает, что пользователь не прошёл аутентификацию.
var ErrInvalidCredentials = errors.New("invalid credentials")

// UserCredentialReader читает пользователя и хэш пароля для аутентификации.
type UserCredentialReader interface {
	// FindByLogin возвращает публичные данные пользователя и хэш пароля.
	FindByLogin(ctx context.Context, login string) (model.User, []byte, error)
}

// PasswordChecker проверяет пароль по сохранённому хэшу.
type PasswordChecker interface {
	// Check проверяет соответствие пароля переданному хэшу.
	Check(password string, hash []byte) error
}

// TokenIssuer выпускает bearer token для аутентифицированного пользователя.
type TokenIssuer interface {
	// Issue выпускает token для пользователя и возвращает срок его действия.
	Issue(ctx context.Context, userID int64) (string, time.Time, error)
}

// AuthenticationResult содержит результат успешной аутентификации.
type AuthenticationResult struct {
	// User содержит публичные данные аутентифицированного пользователя.
	User model.User

	// AccessToken содержит выпущенный bearer token.
	AccessToken string

	// ExpiresAt содержит время истечения bearer token.
	ExpiresAt time.Time
}

// AuthenticationService реализует сценарий аутентификации пользователя.
type AuthenticationService struct {
	users     UserCredentialReader
	passwords PasswordChecker
	tokens    TokenIssuer
}

// NewAuthenticationService создаёт сервис аутентификации пользователя.
func NewAuthenticationService(
	users UserCredentialReader,
	passwords PasswordChecker,
	tokens TokenIssuer,
) *AuthenticationService {
	return &AuthenticationService{
		users:     users,
		passwords: passwords,
		tokens:    tokens,
	}
}

// Authenticate проверяет login и password и выпускает token для пользователя.
func (s *AuthenticationService) Authenticate(
	ctx context.Context,
	login string,
	password string,
) (AuthenticationResult, error) {
	canonicalLogin, err := validateCredentials(login, password)
	if err != nil {
		return AuthenticationResult{}, ErrInvalidCredentials
	}

	if err := checkContext(ctx); err != nil {
		return AuthenticationResult{}, err
	}

	user, passwordHash, err := s.users.FindByLogin(ctx, canonicalLogin)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return AuthenticationResult{}, ErrInvalidCredentials
		}

		return AuthenticationResult{}, fmt.Errorf("find user by login: %w", err)
	}

	if err := checkContext(ctx); err != nil {
		return AuthenticationResult{}, err
	}

	if err := s.passwords.Check(password, passwordHash); err != nil {
		if err := checkContext(ctx); err != nil {
			return AuthenticationResult{}, err
		}

		return AuthenticationResult{}, ErrInvalidCredentials
	}

	if err := checkContext(ctx); err != nil {
		return AuthenticationResult{}, err
	}

	accessToken, expiresAt, err := s.tokens.Issue(ctx, user.ID)
	if err != nil {
		return AuthenticationResult{}, fmt.Errorf("issue token: %w", err)
	}

	return AuthenticationResult{
		User:        user,
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
	}, nil
}

func checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	return nil
}
