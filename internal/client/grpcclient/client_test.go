package grpcclient

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestNew(t *testing.T) {
	client, err := New("localhost:9090", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNew_RejectsEmptyAddress(t *testing.T) {
	_, err := New("   ", "")
	if !errors.Is(err, errAddressRequired) {
		t.Fatalf("New() error = %v, want %v", err, errAddressRequired)
	}
}

func TestClient_CloseNil(t *testing.T) {
	var client *Client
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClient_Close(t *testing.T) {
	closer := &closerStub{}
	client := &Client{connection: closer}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !closer.called {
		t.Fatal("Close() did not close connection")
	}
}

func TestClient_CloseReturnsError(t *testing.T) {
	wantErr := errors.New("close failed")
	client := &Client{connection: &closerStub{err: wantErr}}
	if err := client.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("Close() error = %v, want %v", err, wantErr)
	}
}

func TestClient_Health(t *testing.T) {
	client := &Client{health: healthClientStub{check: func(
		_ context.Context,
		request *healthpb.HealthCheckRequest,
		_ ...grpc.CallOption,
	) (*healthpb.HealthCheckResponse, error) {
		if request.GetService() != "" {
			t.Fatalf("Health service = %q, want empty", request.GetService())
		}
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
	}}}

	status, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if status != "ok" {
		t.Fatalf("Health() status = %q, want ok", status)
	}
}

func TestClient_HealthReturnsUnavailableForNonServingStatus(t *testing.T) {
	client := &Client{health: healthClientStub{check: func(
		context.Context,
		*healthpb.HealthCheckRequest,
		...grpc.CallOption,
	) (*healthpb.HealthCheckResponse, error) {
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_NOT_SERVING}, nil
	}}}

	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("Health() error = nil, want unavailable error")
	}
	if got := failure.KindOf(err); got != failure.Unavailable {
		t.Fatalf("failure.KindOf() = %d, want %d", got, failure.Unavailable)
	}
}

func TestClient_HealthMapsRPCError(t *testing.T) {
	client := &Client{health: healthClientStub{check: func(
		context.Context,
		*healthpb.HealthCheckRequest,
		...grpc.CallOption,
	) (*healthpb.HealthCheckResponse, error) {
		return nil, status.Error(codes.Unavailable, "connection refused")
	}}}

	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("Health() error = nil, want RPC error")
	}
	if got := failure.KindOf(err); got != failure.Unavailable {
		t.Fatalf("failure.KindOf() = %d, want %d", got, failure.Unavailable)
	}
}

func TestClient_HealthRejectsNilResponse(t *testing.T) {
	client := &Client{health: healthClientStub{check: func(
		context.Context,
		*healthpb.HealthCheckRequest,
		...grpc.CallOption,
	) (*healthpb.HealthCheckResponse, error) {
		return nil, nil
	}}}

	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("Health() error = nil, want invalid response")
	}
	if got := failure.Message(err); got != "Invalid server response" {
		t.Fatalf("failure.Message() = %q", got)
	}
}
