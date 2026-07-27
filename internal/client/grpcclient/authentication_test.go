package grpcclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestClient_Register(t *testing.T) {
	client := &Client{auth: authClientStub{register: func(
		_ context.Context,
		request *gopherkeeperpb.RegisterRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.RegisterResponse, error) {
		if request.GetLogin() != "Alice" || request.GetPassword() != "secret" {
			t.Fatalf("Register request = %q / %q", request.GetLogin(), request.GetPassword())
		}
		response := &gopherkeeperpb.RegisterResponse{}
		response.SetUser(protoUser())
		return response, nil
	}}}

	user, err := client.Register(context.Background(), "Alice", "secret")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.ID != 42 || user.Login != "alice" || !user.CreatedAt.Equal(testTime) {
		t.Fatalf("Register() user = %+v", user)
	}
}

func TestClient_RegisterMapsAlreadyExists(t *testing.T) {
	client := &Client{auth: authClientStub{register: func(
		context.Context,
		*gopherkeeperpb.RegisterRequest,
		...grpc.CallOption,
	) (*gopherkeeperpb.RegisterResponse, error) {
		return nil, status.Error(codes.AlreadyExists, "login is already registered")
	}}}

	_, err := client.Register(context.Background(), "alice", "secret")
	if !errors.Is(err, model.ErrLoginAlreadyExists) {
		t.Fatalf("Register() error = %v, want login exists", err)
	}
}

func TestClient_Login(t *testing.T) {
	client := &Client{auth: authClientStub{login: func(
		_ context.Context,
		request *gopherkeeperpb.LoginRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.LoginResponse, error) {
		if request.GetLogin() != "Alice" || request.GetPassword() != "secret" {
			t.Fatalf("Login request = %q / %q", request.GetLogin(), request.GetPassword())
		}
		response := &gopherkeeperpb.LoginResponse{}
		response.SetUser(protoUser())
		response.SetAccessToken("token")
		response.SetExpiresAt(timestamppb.New(testTime.Add(time.Hour)))
		return response, nil
	}}}

	authentication, err := client.Login(context.Background(), "Alice", "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if authentication.AccessToken != "token" || authentication.User.Login != "alice" {
		t.Fatalf("Login() result = %+v", authentication)
	}
	if !authentication.ExpiresAt.Equal(testTime.Add(time.Hour)) {
		t.Fatalf("Login() expires at = %v", authentication.ExpiresAt)
	}
}

func TestClient_LoginMapsInvalidCredentials(t *testing.T) {
	client := &Client{auth: authClientStub{login: func(
		context.Context,
		*gopherkeeperpb.LoginRequest,
		...grpc.CallOption,
	) (*gopherkeeperpb.LoginResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "invalid login or password")
	}}}

	_, err := client.Login(context.Background(), "alice", "wrong")
	if !errors.Is(err, model.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want invalid credentials", err)
	}
}

func TestClient_CurrentUserSendsBearerToken(t *testing.T) {
	client := &Client{auth: authClientStub{currentUser: func(
		ctx context.Context,
		_ *gopherkeeperpb.CurrentUserRequest,
		_ ...grpc.CallOption,
	) (*gopherkeeperpb.CurrentUserResponse, error) {
		assertBearerToken(t, ctx, "token")
		response := &gopherkeeperpb.CurrentUserResponse{}
		response.SetUser(protoUser())
		return response, nil
	}}}

	user, err := client.CurrentUser(context.Background(), "token")
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if user.Login != "alice" {
		t.Fatalf("CurrentUser() login = %q", user.Login)
	}
}

func TestClient_CurrentUserMapsUnauthorized(t *testing.T) {
	client := &Client{auth: authClientStub{currentUser: func(
		context.Context,
		*gopherkeeperpb.CurrentUserRequest,
		...grpc.CallOption,
	) (*gopherkeeperpb.CurrentUserResponse, error) {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}}}

	_, err := client.CurrentUser(context.Background(), "token")
	if !errors.Is(err, model.ErrUnauthorized) {
		t.Fatalf("CurrentUser() error = %v, want unauthorized", err)
	}
}

func TestClient_LoginRejectsMalformedResponse(t *testing.T) {
	client := &Client{auth: authClientStub{login: func(
		context.Context,
		*gopherkeeperpb.LoginRequest,
		...grpc.CallOption,
	) (*gopherkeeperpb.LoginResponse, error) {
		return &gopherkeeperpb.LoginResponse{}, nil
	}}}

	_, err := client.Login(context.Background(), "alice", "secret")
	if err == nil {
		t.Fatal("Login() error = nil, want invalid response")
	}
}
