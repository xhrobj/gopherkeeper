package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRegistrationPassword = "correct-horse-battery-staple"

func TestRegistrationService_Register(t *testing.T) {
	passwordHash := []byte("prepared-password-hash")
	createdAt := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	passwords := &passwordHasherStub{
		hashFunc: func(gotPassword string) ([]byte, error) {
			if gotPassword != testRegistrationPassword {
				t.Error("Hash() password differs from registration password")
			}

			return passwordHash, nil
		},
	}

	users := &userRepositoryStub{
		createFunc: func(
			_ context.Context,
			login string,
			gotHash []byte,
		) (model.User, error) {
			if login != "alice" {
				t.Errorf("Create() login = %q, want %q", login, "alice")
			}

			if !bytes.Equal(gotHash, passwordHash) {
				t.Errorf(
					"Create() passwordHash = %q, want %q",
					gotHash,
					passwordHash,
				)
			}

			if bytes.Equal(gotHash, []byte(testRegistrationPassword)) {
				t.Error("Create() received plaintext password")
			}

			return model.User{
				ID:        42,
				Login:     login,
				CreatedAt: createdAt,
			}, nil
		},
	}

	service := NewRegistrationService(users, passwords)

	user, err := service.Register(
		context.Background(),
		" Alice ",
		testRegistrationPassword,
	)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	want := model.User{
		ID:        42,
		Login:     "alice",
		CreatedAt: createdAt,
	}
	if user != want {
		t.Errorf("Register() user = %+v, want %+v", user, want)
	}

	if passwords.calls != 1 {
		t.Errorf("Hash() calls = %d, want 1", passwords.calls)
	}

	if users.calls != 1 {
		t.Errorf("Create() calls = %d, want 1", users.calls)
	}
}

func TestRegistrationService_RegisterValidationError(t *testing.T) {
	passwords := &passwordHasherStub{}
	users := &userRepositoryStub{}
	service := NewRegistrationService(users, passwords)

	_, err := service.Register(
		context.Background(),
		".eve",
		testRegistrationPassword,
	)
	if !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("Register() error = %v, want %v", err, ErrInvalidLogin)
	}

	if passwords.calls != 0 {
		t.Errorf("Hash() calls = %d, want 0", passwords.calls)
	}

	if users.calls != 0 {
		t.Errorf("Create() calls = %d, want 0", users.calls)
	}
}

func TestRegistrationService_RegisterHashError(t *testing.T) {
	hashErr := errors.New("hash failed")

	passwords := &passwordHasherStub{
		hashFunc: func(string) ([]byte, error) {
			return nil, hashErr
		},
	}
	users := &userRepositoryStub{}
	service := NewRegistrationService(users, passwords)

	_, err := service.Register(
		context.Background(),
		"eve",
		testRegistrationPassword,
	)
	if !errors.Is(err, hashErr) {
		t.Fatalf("Register() error = %v, want wrapped %v", err, hashErr)
	}

	if passwords.calls != 1 {
		t.Errorf("Hash() calls = %d, want 1", passwords.calls)
	}

	if users.calls != 0 {
		t.Errorf("Create() calls = %d, want 0", users.calls)
	}
}

func TestRegistrationService_RegisterRepositoryError(t *testing.T) {
	passwords := &passwordHasherStub{
		hashFunc: func(string) ([]byte, error) {
			return []byte("prepared-password-hash"), nil
		},
	}
	users := &userRepositoryStub{
		createFunc: func(
			context.Context,
			string,
			[]byte,
		) (model.User, error) {
			return model.User{}, model.ErrLoginAlreadyExists
		},
	}
	service := NewRegistrationService(users, passwords)

	_, err := service.Register(
		context.Background(),
		"eve",
		testRegistrationPassword,
	)
	if !errors.Is(err, model.ErrLoginAlreadyExists) {
		t.Fatalf(
			"Register() error = %v, want wrapped %v",
			err,
			model.ErrLoginAlreadyExists,
		)
	}

	if passwords.calls != 1 {
		t.Errorf("Hash() calls = %d, want 1", passwords.calls)
	}

	if users.calls != 1 {
		t.Errorf("Create() calls = %d, want 1", users.calls)
	}
}

type passwordHasherStub struct {
	hashFunc func(password string) ([]byte, error)
	calls    int
}

func (s *passwordHasherStub) Hash(password string) ([]byte, error) {
	s.calls++

	if s.hashFunc == nil {
		return nil, errors.New("unexpected Hash call")
	}

	return s.hashFunc(password)
}

type userRepositoryStub struct {
	createFunc func(
		ctx context.Context,
		login string,
		passwordHash []byte,
	) (model.User, error)
	calls int
}

func (s *userRepositoryStub) Create(
	ctx context.Context,
	login string,
	passwordHash []byte,
) (model.User, error) {
	s.calls++

	if s.createFunc == nil {
		return model.User{}, errors.New("unexpected Create call")
	}

	return s.createFunc(ctx, login, passwordHash)
}
