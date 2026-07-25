package grpcserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const shutdownTimeout = 30 * time.Second

// NewServer создаёт TLS-защищённый gRPC-сервер с application services и health service.
func NewServer(
	certFile, keyFile string,
	deps Dependencies,
	logger *zap.Logger,
) (*grpc.Server, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load gRPC TLS credentials: %w", err)
	}
	transportCredentials := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
	})

	server := grpc.NewServer(
		grpc.Creds(transportCredentials),
		grpc.MaxRecvMsgSize(int(model.HTTPRequestBodyMaxSize)),
		grpc.MaxSendMsgSize(int(model.HTTPRequestBodyMaxSize)),
		grpc.ChainUnaryInterceptor(
			loggingUnaryInterceptor(logger),
			authenticationUnaryInterceptor(deps.TokenValidator),
		),
	)
	gopherkeeperpb.RegisterAuthServiceServer(server, newAuthService(deps))
	gopherkeeperpb.RegisterRecordServiceServer(server, newRecordService(deps.Records))

	healthServer := health.NewServer()
	for _, serviceName := range []string{
		"",
		gopherkeeperpb.AuthService_ServiceDesc.ServiceName,
		gopherkeeperpb.RecordService_ServiceDesc.ServiceName,
	} {
		healthServer.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
	}
	healthpb.RegisterHealthServer(server, healthServer)

	return server, nil
}

// Serve запускает gRPC-сервер и корректно останавливает его при отмене контекста.
func Serve(ctx context.Context, address string, server *grpc.Server) (returnErr error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen gRPC on %s: %w", address, err)
	}
	defer func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			returnErr = errors.Join(returnErr, fmt.Errorf("close gRPC listener: %w", err))
		}
	}()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		return ignoreServerStopped(err)
	case <-ctx.Done():
		if err := stop(ctx, server); err != nil {
			return err
		}

		return ignoreServerStopped(<-serveErr)
	}
}

func stop(parent context.Context, server *grpc.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), shutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		return nil
	case <-shutdownCtx.Done():
		server.Stop()
		return fmt.Errorf("shutdown gRPC server: %w", shutdownCtx.Err())
	}
}

func ignoreServerStopped(err error) error {
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}

	return fmt.Errorf("serve gRPC: %w", err)
}
