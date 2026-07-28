package grpcclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/apilimits"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/client/transporttls"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/grpclimits"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const requestTimeout = 10 * time.Second

type connectionCloser interface {
	Close() error
}

type healthChecker interface {
	Check(context.Context, *healthpb.HealthCheckRequest, ...grpc.CallOption) (*healthpb.HealthCheckResponse, error)
}

type authClient interface {
	Register(context.Context, *gopherkeeperpb.RegisterRequest, ...grpc.CallOption) (*gopherkeeperpb.RegisterResponse, error)
	Login(context.Context, *gopherkeeperpb.LoginRequest, ...grpc.CallOption) (*gopherkeeperpb.LoginResponse, error)
	CurrentUser(context.Context, *gopherkeeperpb.CurrentUserRequest, ...grpc.CallOption) (*gopherkeeperpb.CurrentUserResponse, error)
}

type recordClient interface {
	CreateRecord(context.Context, *gopherkeeperpb.CreateRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.Record, error)
	ListRecords(context.Context, *gopherkeeperpb.ListRecordsRequest, ...grpc.CallOption) (*gopherkeeperpb.ListRecordsResponse, error)
	GetRecord(context.Context, *gopherkeeperpb.GetRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.Record, error)
	UpdateRecord(context.Context, *gopherkeeperpb.UpdateRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.Record, error)
	DeleteRecord(context.Context, *gopherkeeperpb.DeleteRecordRequest, ...grpc.CallOption) (*gopherkeeperpb.DeleteRecordResponse, error)
}

// Client выполняет gRPC-вызовы к Серверу GophKeeper.
type Client struct {
	connection connectionCloser
	health     healthChecker
	auth       authClient
	records    recordClient
}

var errAddressRequired = errors.New("gRPC address is required")

var (
	_ io.Closer             = (*Client)(nil)
	_ usecase.HealthChecker = (*Client)(nil)
	_ usecase.UserGateway   = (*Client)(nil)
	_ usecase.RecordGateway = (*Client)(nil)
)

// New создаёт TLS-защищённый gRPC-Клиент с системными корневыми
// сертификатами и дополнительным доверенным CA certificate при его наличии.
func New(address, caCertFile string) (*Client, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errAddressRequired
	}

	tlsConfig, err := transporttls.NewConfig(caCertFile)
	if err != nil {
		return nil, err
	}

	connection, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(apilimits.RequestMaxSize),
			grpc.MaxCallRecvMsgSize(grpclimits.ResponseMessageMaxSize),
		),
	)
	if err != nil {
		return nil, failure.Network("create gRPC client", err)
	}

	return &Client{
		connection: connection,
		health:     healthpb.NewHealthClient(connection),
		auth:       gopherkeeperpb.NewAuthServiceClient(connection),
		records:    gopherkeeperpb.NewRecordServiceClient(connection),
	}, nil
}

// Close закрывает клиентское gRPC-соединение.
func (c *Client) Close() error {
	if c == nil || c.connection == nil {
		return nil
	}
	if err := c.connection.Close(); err != nil {
		return fmt.Errorf("close gRPC connection: %w", err)
	}
	return nil
}

// Health проверяет доступность Сервера и возвращает его технический статус.
func (c *Client) Health(ctx context.Context) (string, error) {
	callCtx, cancel := withRequestTimeout(ctx)
	defer cancel()

	response, err := c.health.Check(callCtx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return "", mapRPCError("health", err, nil)
	}
	if response == nil {
		return "", invalidResponseError("health", errors.New("response is nil"))
	}
	if response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return "", failure.Wrap(
			failure.Unavailable,
			fmt.Sprintf("health status is %s", response.GetStatus()),
			"Server health check failed",
			nil,
		)
	}

	return "ok", nil
}

func withRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, requestTimeout)
}

func authorizedContext(ctx context.Context, accessToken string) context.Context {
	return withBearerToken(ctx, accessToken)
}
