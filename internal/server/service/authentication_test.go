package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testAuthenticationPassword = "correct-horse-battery-staple"

func TestAuthenticationService_Authenticate(t *testing.T) {
	createdAt := time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 3, 12, 15, 0, 0, time.UTC)
	passwordHash := []byte("stored-password-hash")

	users := &userCredentialReaderStub{
		findByLoginFunc: func(
			_ context.Context,
			login string,
		) (model.User, []byte, error) {
			if login != "alice" {
				t.Errorf("FindByLogin() login = %q, want %q", login, "alice")
			}

			return model.User{
				ID:        42,
				Login:     login,
				CreatedAt: createdAt,
			}, passwordHash, nil
		},
	}
	passwords := &passwordCheckerStub{
		checkFunc: func(gotPassword string, gotHash []byte) error {
			if gotPassword != testAuthenticationPassword {
				t.Errorf(
					"Check() password = %q, want %q",
					gotPassword,
					testAuthenticationPassword,
				)
			}

			if !bytes.Equal(gotHash, passwordHash) {
				t.Errorf("Check() hash = %q, want %q", gotHash, passwordHash)
			}

			return nil
		},
	}
	tokens := &tokenIssuerStub{
		issueFunc: func(_ context.Context, userID int64) (string, time.Time, error) {
			if userID != 42 {
				t.Errorf("Issue() userID = %d, want 42", userID)
			}

			return "access-token", expiresAt, nil
		},
	}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	result, err := service.Authenticate(
		context.Background(),
		" Alice ",
		testAuthenticationPassword,
	)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	want := AuthenticationResult{
		User: model.User{
			ID:        42,
			Login:     "alice",
			CreatedAt: createdAt,
		},
		AccessToken: "access-token",
		ExpiresAt:   expiresAt,
	}
	if result != want {
		t.Errorf("Authenticate() result = %+v, want %+v", result, want)
	}

	if users.calls != 1 {
		t.Errorf("FindByLogin() calls = %d, want 1", users.calls)
	}

	if passwords.calls != 1 {
		t.Errorf("Check() calls = %d, want 1", passwords.calls)
	}

	if tokens.calls != 1 {
		t.Errorf("Issue() calls = %d, want 1", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticateInvalidInput(t *testing.T) {
	users := &userCredentialReaderStub{}
	passwords := &passwordCheckerStub{}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(
		context.Background(),
		".eve",
		testAuthenticationPassword,
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"Authenticate() error = %v, want %v",
			err,
			ErrInvalidCredentials,
		)
	}

	if users.calls != 0 {
		t.Errorf("FindByLogin() calls = %d, want 0", users.calls)
	}

	if passwords.calls != 0 {
		t.Errorf("Check() calls = %d, want 0", passwords.calls)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticateMissingUser(t *testing.T) {
	users := &userCredentialReaderStub{
		findByLoginFunc: func(
			context.Context,
			string,
		) (model.User, []byte, error) {
			return model.User{}, nil, model.ErrUserNotFound
		},
	}
	passwords := &passwordCheckerStub{}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(users, passwords, tokens)

	_, err := service.Authenticate(
		context.Background(),
		"eve",
		testAuthenticationPassword,
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"Authenticate() error = %v, want %v",
			err,
			ErrInvalidCredentials,
		)
	}

	if users.calls != 1 {
		t.Errorf("FindByLogin() calls = %d, want 1", users.calls)
	}

	if passwords.calls != 0 {
		t.Errorf("Check() calls = %d, want 0", passwords.calls)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticatePasswordMismatch(t *testing.T) {
	users := successfulUserCredentialReaderStub(t)
	passwords := &passwordCheckerStub{
		checkFunc: func(string, []byte) error {
			return errors.New("password mismatch")
		},
	}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(
		context.Background(),
		"alice",
		testAuthenticationPassword,
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"Authenticate() error = %v, want %v",
			err,
			ErrInvalidCredentials,
		)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticateRepositoryError(t *testing.T) {
	repositoryErr := errors.New("repository failed")
	users := &userCredentialReaderStub{
		findByLoginFunc: func(
			context.Context,
			string,
		) (model.User, []byte, error) {
			return model.User{}, nil, repositoryErr
		},
	}
	passwords := &passwordCheckerStub{}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(
		context.Background(),
		"alice",
		testAuthenticationPassword,
	)
	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"Authenticate() error = %v, want wrapped %v",
			err,
			repositoryErr,
		)
	}

	if passwords.calls != 0 {
		t.Errorf("Check() calls = %d, want 0", passwords.calls)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticatePasswordCheckError(t *testing.T) {
	users := successfulUserCredentialReaderStub(t)
	passwords := &passwordCheckerStub{
		checkFunc: func(string, []byte) error {
			return errors.New("password check failed")
		},
	}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(
		context.Background(),
		"alice",
		testAuthenticationPassword,
	)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"Authenticate() error = %v, want %v",
			err,
			ErrInvalidCredentials,
		)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticateTokenError(t *testing.T) {
	tokenErr := errors.New("token failed")
	users := successfulUserCredentialReaderStub(t)
	passwords := &passwordCheckerStub{
		checkFunc: func(string, []byte) error {
			return nil
		},
	}
	tokens := &tokenIssuerStub{
		issueFunc: func(context.Context, int64) (string, time.Time, error) {
			return "", time.Time{}, tokenErr
		},
	}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(
		context.Background(),
		"alice",
		testAuthenticationPassword,
	)
	if !errors.Is(err, tokenErr) {
		t.Fatalf(
			"Authenticate() error = %v, want wrapped %v",
			err,
			tokenErr,
		)
	}
}

func TestAuthenticationService_AuthenticateCanceledBeforePasswordCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	users := successfulUserCredentialReaderStub(t)
	passwords := &passwordCheckerStub{}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(ctx, "alice", testAuthenticationPassword)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Authenticate() error = %v, want %v", err, context.Canceled)
	}

	if users.calls != 0 {
		t.Errorf("FindByLogin() calls = %d, want 0", users.calls)
	}

	if passwords.calls != 0 {
		t.Errorf("Check() calls = %d, want 0", passwords.calls)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func TestAuthenticationService_AuthenticateCanceledAfterPasswordCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	users := successfulUserCredentialReaderStub(t)
	passwords := &passwordCheckerStub{
		checkFunc: func(string, []byte) error {
			cancel()

			return nil
		},
	}
	tokens := &tokenIssuerStub{}
	service := NewAuthenticationService(
		users,
		passwords,
		tokens,
	)

	_, err := service.Authenticate(ctx, "alice", testAuthenticationPassword)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Authenticate() error = %v, want %v", err, context.Canceled)
	}

	if tokens.calls != 0 {
		t.Errorf("Issue() calls = %d, want 0", tokens.calls)
	}
}

func successfulUserCredentialReaderStub(t *testing.T) *userCredentialReaderStub {
	t.Helper()

	return &userCredentialReaderStub{
		findByLoginFunc: func(
			context.Context,
			string,
		) (model.User, []byte, error) {
			return model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC),
			}, []byte("stored-password-hash"), nil
		},
	}
}

type userCredentialReaderStub struct {
	findByLoginFunc func(
		ctx context.Context,
		login string,
	) (model.User, []byte, error)
	calls int
}

func (s *userCredentialReaderStub) FindByLogin(
	ctx context.Context,
	login string,
) (model.User, []byte, error) {
	s.calls++

	if s.findByLoginFunc == nil {
		return model.User{}, nil, errors.New("unexpected FindByLogin call")
	}

	return s.findByLoginFunc(ctx, login)
}

type passwordCheckerStub struct {
	checkFunc func(password string, hash []byte) error
	calls     int
}

func (s *passwordCheckerStub) Check(password string, hash []byte) error {
	s.calls++

	if s.checkFunc == nil {
		return errors.New("unexpected Check call")
	}

	return s.checkFunc(password, hash)
}

type tokenIssuerStub struct {
	issueFunc func(ctx context.Context, userID int64) (string, time.Time, error)
	calls     int
}

func (s *tokenIssuerStub) Issue(
	ctx context.Context,
	userID int64,
) (string, time.Time, error) {
	s.calls++

	if s.issueFunc == nil {
		return "", time.Time{}, errors.New("unexpected Issue call")
	}

	return s.issueFunc(ctx, userID)
}
