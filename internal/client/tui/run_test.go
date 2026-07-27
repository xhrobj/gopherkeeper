package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestRun_RejectsMissingBackendFactory(t *testing.T) {
	err := Run(context.Background(), Options{})
	if err == nil || !strings.Contains(err.Error(), "backend factory is required") {
		t.Fatalf("Run() error = %v, want missing backend factory error", err)
	}
}

func TestRun_ReturnsBackendFactoryError(t *testing.T) {
	want := errors.New("backend creation failed")
	err := Run(context.Background(), Options{
		BackendFactory: func(config.Config) (Backend, error) {
			return nil, want
		},
	})
	if !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want %v", err, want)
	}
}

func TestRun_RejectsNilBackend(t *testing.T) {
	err := Run(context.Background(), Options{
		BackendFactory: func(config.Config) (Backend, error) {
			return nil, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "factory returned nil backend") {
		t.Fatalf("Run() error = %v, want nil backend error", err)
	}
}

func TestCloseFinalBackend_UsesCurrentBackend(t *testing.T) {
	fallbackCloseCalls := 0
	currentCloseCalls := 0
	fallback := backendStub{closeCache: func() { fallbackCloseCalls++ }}
	current := backendStub{closeCache: func() { currentCloseCalls++ }}

	closeFinalBackend(model{backend: current}, fallback)
	if currentCloseCalls != 1 || fallbackCloseCalls != 0 {
		t.Fatalf("cache close calls = current %d fallback %d", currentCloseCalls, fallbackCloseCalls)
	}

	closeFinalBackend(nil, fallback)
	if fallbackCloseCalls != 1 {
		t.Fatalf("fallback cache close calls = %d, want 1", fallbackCloseCalls)
	}
}

func TestCloseFinalBackend_ClosesCacheAndTransport(t *testing.T) {
	cacheCloseCalls := 0
	transportCloseCalls := 0
	backend := &transportClosingBackendStub{
		backendStub: backendStub{closeCache: func() { cacheCloseCalls++ }},
		closeTransport: func() error {
			transportCloseCalls++
			return nil
		},
	}

	closeFinalBackend(model{backend: backend}, nil)

	if cacheCloseCalls != 1 || transportCloseCalls != 1 {
		t.Fatalf("close calls = cache %d transport %d, want 1/1", cacheCloseCalls, transportCloseCalls)
	}
}
