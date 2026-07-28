package grpcserver

import (
	"context"
	"time"

	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const healthCheckTimeout = 2 * time.Second

type healthService struct {
	healthpb.UnimplementedHealthServer
	database DatabasePinger
}

var healthServiceNames = []string{
	"",
	gopherkeeperpb.AuthService_ServiceDesc.ServiceName,
	gopherkeeperpb.RecordService_ServiceDesc.ServiceName,
}

func (service *healthService) Check(
	ctx context.Context,
	request *healthpb.HealthCheckRequest,
) (*healthpb.HealthCheckResponse, error) {
	if !isKnownHealthService(request.GetService()) {
		return nil, status.Error(codes.NotFound, "health service is not registered")
	}

	return &healthpb.HealthCheckResponse{
		Status: service.servingStatus(ctx),
	}, nil
}

func (service *healthService) List(
	ctx context.Context,
	_ *healthpb.HealthListRequest,
) (*healthpb.HealthListResponse, error) {
	servingStatus := service.servingStatus(ctx)
	statuses := make(map[string]*healthpb.HealthCheckResponse, len(healthServiceNames))
	for _, serviceName := range healthServiceNames {
		statuses[serviceName] = &healthpb.HealthCheckResponse{Status: servingStatus}
	}

	return &healthpb.HealthListResponse{Statuses: statuses}, nil
}

func (*healthService) Watch(
	*healthpb.HealthCheckRequest,
	grpc.ServerStreamingServer[healthpb.HealthCheckResponse],
) error {
	return status.Error(codes.Unimplemented, "health watch is not supported")
}

func newHealthService(database DatabasePinger) *healthService {
	return &healthService{database: database}
}

func (service *healthService) servingStatus(ctx context.Context) healthpb.HealthCheckResponse_ServingStatus {
	pingCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	if service.database == nil || service.database.Ping(pingCtx) != nil {
		return healthpb.HealthCheckResponse_NOT_SERVING
	}

	return healthpb.HealthCheckResponse_SERVING
}

func isKnownHealthService(serviceName string) bool {
	for _, knownServiceName := range healthServiceNames {
		if serviceName == knownServiceName {
			return true
		}
	}

	return false
}
