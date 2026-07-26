package grpcserver

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthService_Register(t *testing.T) {
	createdAt := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	server := newAuthService(Dependencies{
		Registerer: userRegistererFunc(func(_ context.Context, login, password string) (model.User, error) {
			if login != "Alice" || password != "correct-horse-battery-staple" {
				t.Fatalf("Register() credentials = %q / %q", login, password)
			}
			return model.User{ID: 42, Login: "alice", CreatedAt: createdAt}, nil
		}),
	})

	response, err := server.Register(
		context.Background(),
		newProtoRegisterRequest("Alice", "correct-horse-battery-staple"),
	)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if response.GetUser().GetId() != 42 || response.GetUser().GetLogin() != "alice" {
		t.Fatalf("Register() user = %#v", response.GetUser())
	}
}

func TestAuthService_Login(t *testing.T) {
	createdAt := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	expiresAt := createdAt.Add(time.Hour)
	server := newAuthService(Dependencies{
		Authenticator: userAuthenticatorFunc(func(_ context.Context, login, password string) (service.AuthenticationResult, error) {
			if login != "alice" || password != "correct-horse-battery-staple" {
				t.Fatalf("Authenticate() credentials = %q / %q", login, password)
			}
			return service.AuthenticationResult{
				User:        model.User{ID: 42, Login: "alice", CreatedAt: createdAt},
				AccessToken: "test.jwt.token",
				ExpiresAt:   expiresAt,
			}, nil
		}),
	})

	response, err := server.Login(
		context.Background(),
		newProtoLoginRequest("alice", "correct-horse-battery-staple"),
	)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if response.GetAccessToken() != "test.jwt.token" || !response.GetExpiresAt().AsTime().Equal(expiresAt) {
		t.Fatalf("Login() response = %#v", response)
	}
}

func TestAuthService_CurrentUser(t *testing.T) {
	server := newAuthService(Dependencies{
		CurrentUserReader: currentUserReaderFunc(func(_ context.Context, id int64) (model.User, error) {
			if id != 42 {
				t.Fatalf("FindByID() id = %d, want 42", id)
			}
			return model.User{
				ID:        id,
				Login:     "alice",
				CreatedAt: time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC),
			}, nil
		}),
	})
	ctx := context.WithValue(context.Background(), userIDContextKey{}, int64(42))

	response, err := server.CurrentUser(ctx, &gopherkeeperpb.CurrentUserRequest{})
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if response.GetUser().GetLogin() != "alice" {
		t.Fatalf("CurrentUser() login = %q", response.GetUser().GetLogin())
	}
}

func TestAuthService_RejectsInvalidCalls(t *testing.T) {
	server := newAuthService(Dependencies{})

	if _, err := server.Register(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Register() code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if _, err := server.Login(context.Background(), &gopherkeeperpb.LoginRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("Login() code = %s, want %s", status.Code(err), codes.Internal)
	}
	if _, err := server.CurrentUser(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("CurrentUser(nil) code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
	if _, err := server.CurrentUser(context.Background(), &gopherkeeperpb.CurrentUserRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("CurrentUser() code = %s, want %s", status.Code(err), codes.Unauthenticated)
	}
}
