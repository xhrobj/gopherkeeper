package grpcserver

import (
	"context"
	"errors"
	"testing"

	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthenticationUnaryInterceptor_AllowsPublicMethod(t *testing.T) {
	called := false
	interceptor := authenticationUnaryInterceptor(nil)

	_, err := interceptor(
		context.Background(),
		&gopherkeeperpb.LoginRequest{},
		&grpc.UnaryServerInfo{FullMethod: gopherkeeperpb.AuthService_Login_FullMethodName},
		func(context.Context, any) (any, error) {
			called = true
			return &gopherkeeperpb.LoginResponse{}, nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if !called {
		t.Fatal("public handler was not called")
	}
}

func TestAuthenticationUnaryInterceptor_AllowsHealthMethods(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		request    any
	}{
		{
			name:       "check",
			fullMethod: healthpb.Health_Check_FullMethodName,
			request:    &healthpb.HealthCheckRequest{},
		},
		{
			name:       "list",
			fullMethod: healthpb.Health_List_FullMethodName,
			request:    &healthpb.HealthListRequest{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			interceptor := authenticationUnaryInterceptor(nil)

			_, err := interceptor(
				context.Background(),
				test.request,
				&grpc.UnaryServerInfo{FullMethod: test.fullMethod},
				func(context.Context, any) (any, error) {
					called = true
					return new(struct{}), nil
				},
			)
			if err != nil {
				t.Fatalf("interceptor() error = %v", err)
			}
			if !called {
				t.Fatal("health handler was not called")
			}
		})
	}
}

func TestAuthenticationUnaryInterceptor_AddsUserID(t *testing.T) {
	interceptor := authenticationUnaryInterceptor(tokenValidatorFunc(
		func(_ context.Context, token string) (int64, error) {
			if token != "valid-token" {
				t.Fatalf("Validate() token = %q", token)
			}
			return 42, nil
		},
	))
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs("authorization", "Bearer valid-token"),
	)

	_, err := interceptor(
		ctx,
		&gopherkeeperpb.CurrentUserRequest{},
		&grpc.UnaryServerInfo{FullMethod: gopherkeeperpb.AuthService_CurrentUser_FullMethodName},
		func(ctx context.Context, _ any) (any, error) {
			userID, ok := userIDFromContext(ctx)
			if !ok || userID != 42 {
				t.Fatalf("userIDFromContext() = %d, %t", userID, ok)
			}
			return &gopherkeeperpb.CurrentUserResponse{}, nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
}

func TestAuthenticationUnaryInterceptor_RejectsInvalidAuthentication(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		validator TokenValidator
	}{
		{name: "missing metadata", ctx: context.Background(), validator: tokenValidatorFunc(func(context.Context, string) (int64, error) { return 42, nil })},
		{name: "invalid scheme", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic token")), validator: tokenValidatorFunc(func(context.Context, string) (int64, error) { return 42, nil })},
		{name: "multiple headers", ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{"authorization": []string{"Bearer one", "Bearer two"}}), validator: tokenValidatorFunc(func(context.Context, string) (int64, error) { return 42, nil })},
		{name: "missing validator", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))},
		{name: "invalid token", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token")), validator: tokenValidatorFunc(func(context.Context, string) (int64, error) { return 0, errors.New("invalid") })},
		{name: "invalid user ID", ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token")), validator: tokenValidatorFunc(func(context.Context, string) (int64, error) { return 0, nil })},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interceptor := authenticationUnaryInterceptor(test.validator)
			_, err := interceptor(
				test.ctx,
				&gopherkeeperpb.CurrentUserRequest{},
				&grpc.UnaryServerInfo{FullMethod: gopherkeeperpb.AuthService_CurrentUser_FullMethodName},
				func(context.Context, any) (any, error) {
					t.Fatal("protected handler must not be called")
					return nil, nil
				},
			)
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("status.Code(error) = %s, want %s", status.Code(err), codes.Unauthenticated)
			}
		})
	}
}
