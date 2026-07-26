package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestServeTransports_StopsAllAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, 2)
	serveErr := make(chan error, 1)

	serve := func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		return nil
	}

	go func() {
		serveErr <- ServeTransports(
			ctx,
			Transport{Name: "HTTPS", Serve: serve},
			Transport{Name: "gRPC", Serve: serve},
		)
	}()

	<-started
	<-started
	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("ServeTransports() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeTransports() did not stop after context cancellation")
	}
}

func TestServeTransports_StopsSiblingAfterTransportError(t *testing.T) {
	wantErr := errors.New("listen failed")
	peerStopped := make(chan struct{})

	err := ServeTransports(
		context.Background(),
		Transport{
			Name: "HTTPS",
			Serve: func(context.Context) error {
				return wantErr
			},
		},
		Transport{
			Name: "gRPC",
			Serve: func(ctx context.Context) error {
				<-ctx.Done()
				close(peerStopped)
				return nil
			},
		},
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("ServeTransports() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "HTTPS transport") {
		t.Fatalf("ServeTransports() error = %q, want transport name", err)
	}
	select {
	case <-peerStopped:
	default:
		t.Fatal("sibling transport was not stopped")
	}
}

func TestServeTransports_RejectsUnexpectedStop(t *testing.T) {
	peerStopped := make(chan struct{})

	err := ServeTransports(
		context.Background(),
		Transport{
			Name: "HTTPS",
			Serve: func(context.Context) error {
				return nil
			},
		},
		Transport{
			Name: "gRPC",
			Serve: func(ctx context.Context) error {
				<-ctx.Done()
				close(peerStopped)
				return nil
			},
		},
	)

	if err == nil || !strings.Contains(err.Error(), "HTTPS transport stopped unexpectedly") {
		t.Fatalf("ServeTransports() error = %v, want unexpected stop error", err)
	}
	select {
	case <-peerStopped:
	default:
		t.Fatal("sibling transport was not stopped")
	}
}

func TestServeTransports_RequiresTransport(t *testing.T) {
	err := ServeTransports(context.Background())
	if err == nil || !strings.Contains(err.Error(), "server transports are required") {
		t.Fatalf("ServeTransports() error = %v, want missing transports error", err)
	}
}
