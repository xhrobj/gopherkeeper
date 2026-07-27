package grpcserver

import (
	"context"
	"errors"
	"testing"

	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestHealthService_Check(t *testing.T) {
	tests := []struct {
		name       string
		service    string
		pingError  error
		wantStatus healthpb.HealthCheckResponse_ServingStatus
	}{
		{
			name:       "overall health available",
			wantStatus: healthpb.HealthCheckResponse_SERVING,
		},
		{
			name:       "registered service available",
			service:    gopherkeeperpb.AuthService_ServiceDesc.ServiceName,
			wantStatus: healthpb.HealthCheckResponse_SERVING,
		},
		{
			name:       "database unavailable",
			service:    gopherkeeperpb.RecordService_ServiceDesc.ServiceName,
			pingError:  errors.New("database unavailable"),
			wantStatus: healthpb.HealthCheckResponse_NOT_SERVING,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newHealthService(databasePingerFunc(func(context.Context) error {
				return test.pingError
			}))

			response, err := service.Check(
				context.Background(),
				&healthpb.HealthCheckRequest{Service: test.service},
			)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			if response.GetStatus() != test.wantStatus {
				t.Fatalf("Check() status = %s, want %s", response.GetStatus(), test.wantStatus)
			}
		})
	}
}

func TestHealthService_CheckRejectsUnknownService(t *testing.T) {
	pingCalls := 0
	service := newHealthService(databasePingerFunc(func(context.Context) error {
		pingCalls++
		return nil
	}))

	_, err := service.Check(
		context.Background(),
		&healthpb.HealthCheckRequest{Service: "unknown.Service"},
	)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("Check() code = %s, want %s", status.Code(err), codes.NotFound)
	}
	if pingCalls != 0 {
		t.Fatalf("Ping() calls = %d, want 0", pingCalls)
	}
}

func TestHealthService_List(t *testing.T) {
	service := newHealthService(databasePingerFunc(func(context.Context) error {
		return errors.New("database unavailable")
	}))

	response, err := service.List(context.Background(), &healthpb.HealthListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(response.GetStatuses()) != len(healthServiceNames) {
		t.Fatalf("List() statuses = %d, want %d", len(response.GetStatuses()), len(healthServiceNames))
	}
	for _, serviceName := range healthServiceNames {
		serviceStatus, ok := response.GetStatuses()[serviceName]
		if !ok {
			t.Fatalf("List() does not contain service %q", serviceName)
		}
		if serviceStatus.GetStatus() != healthpb.HealthCheckResponse_NOT_SERVING {
			t.Fatalf(
				"List() service %q status = %s, want %s",
				serviceName,
				serviceStatus.GetStatus(),
				healthpb.HealthCheckResponse_NOT_SERVING,
			)
		}
	}
}

func TestHealthService_WatchIsNotSupported(t *testing.T) {
	service := newHealthService(databasePingerFunc(func(context.Context) error { return nil }))

	err := service.Watch(&healthpb.HealthCheckRequest{}, nil)
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("Watch() code = %s, want %s", status.Code(err), codes.Unimplemented)
	}
}
