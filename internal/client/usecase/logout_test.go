package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

type sessionDeleterStub struct {
	delete func() error
}

func (s sessionDeleterStub) Delete() error {
	return s.delete()
}

func TestNewLogout(t *testing.T) {
	application, err := NewLogout(config.Config{
		SessionFile: t.TempDir() + "/session.json",
	})
	if err != nil {
		t.Fatalf("NewLogout() error = %v", err)
	}
	if application.sessions == nil {
		t.Error("NewLogout() session deleter = nil")
	}
}

func TestLogoutApplication_Logout(t *testing.T) {
	var deleted bool
	application := &LogoutApplication{
		sessions: sessionDeleterStub{
			delete: func() error {
				deleted = true
				return nil
			},
		},
	}

	if err := application.Logout(context.Background()); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !deleted {
		t.Error("session was not deleted")
	}
}

func TestLogoutApplication_LogoutReturnsDeleteError(t *testing.T) {
	deleteError := errors.New("permission denied")
	application := &LogoutApplication{
		sessions: sessionDeleterStub{
			delete: func() error { return deleteError },
		},
	}

	err := application.Logout(context.Background())
	if err == nil {
		t.Fatal("Logout() error = nil, want delete error")
	}
	if !errors.Is(err, deleteError) {
		t.Error("logout error does not preserve delete error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("logout error contains access token")
	}
}
