package grpcserver

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCTransport_TLSAuthenticationFlow(t *testing.T) {
	certFile, keyFile := grpcTestCertificateFiles(t)
	createdAt := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	deps := completeDependencies()
	deps.Registerer = userRegistererFunc(func(_ context.Context, login, password string) (model.User, error) {
		if login != "Alice" || password != "correct-horse-battery-staple" {
			t.Fatalf("Register() credentials = %q / %q", login, password)
		}
		return model.User{ID: 42, Login: "alice", CreatedAt: createdAt}, nil
	})
	deps.TokenValidator = tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
		if token != "valid-token" {
			t.Fatalf("Validate() token = %q", token)
		}
		return 42, nil
	})
	deps.CurrentUserReader = currentUserReaderFunc(func(_ context.Context, userID int64) (model.User, error) {
		if userID != 42 {
			t.Fatalf("FindByID() userID = %d", userID)
		}
		return model.User{ID: 42, Login: "alice", CreatedAt: createdAt}, nil
	})
	server, err := NewServer(certFile, keyFile, deps, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	address := freeTCPAddress(t)
	serverCtx, cancelServer := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- Serve(serverCtx, address, server)
	}()
	waitForTCPListener(t, address)

	clientCredentials, err := credentials.NewClientTLSFromFile(certFile, "localhost")
	if err != nil {
		cancelServer()
		t.Fatalf("NewClientTLSFromFile() error = %v", err)
	}
	connection, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(clientCredentials),
	)
	if err != nil {
		cancelServer()
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer func() { _ = connection.Close() }()
	defer func() {
		cancelServer()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Errorf("Serve() error = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Serve() did not stop")
		}
	}()

	callCtx, cancelCall := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCall()
	healthResponse, err := healthpb.NewHealthClient(connection).Check(
		callCtx,
		&healthpb.HealthCheckRequest{},
	)
	if err != nil {
		t.Fatalf("Health.Check() error = %v", err)
	}
	if healthResponse.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("Health.Check() status = %s", healthResponse.GetStatus())
	}
	serviceHealth, err := healthpb.NewHealthClient(connection).Check(
		callCtx,
		&healthpb.HealthCheckRequest{Service: gopherkeeperpb.AuthService_ServiceDesc.ServiceName},
	)
	if err != nil {
		t.Fatalf("Health.Check(AuthService) error = %v", err)
	}
	if serviceHealth.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("Health.Check(AuthService) status = %s", serviceHealth.GetStatus())
	}

	authClient := gopherkeeperpb.NewAuthServiceClient(connection)
	registerResponse, err := authClient.Register(
		callCtx,
		newProtoRegisterRequest("Alice", "correct-horse-battery-staple"),
	)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if registerResponse.GetUser().GetLogin() != "alice" {
		t.Fatalf("Register() login = %q", registerResponse.GetUser().GetLogin())
	}

	if _, err := authClient.CurrentUser(callCtx, &gopherkeeperpb.CurrentUserRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("CurrentUser() without token code = %s", status.Code(err))
	}
	authorizedCtx := metadata.AppendToOutgoingContext(
		callCtx,
		"authorization",
		"Bearer valid-token",
	)
	currentUser, err := authClient.CurrentUser(authorizedCtx, &gopherkeeperpb.CurrentUserRequest{})
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if currentUser.GetUser().GetId() != 42 {
		t.Fatalf("CurrentUser() id = %d", currentUser.GetUser().GetId())
	}
}
