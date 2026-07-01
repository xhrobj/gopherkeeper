package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOpenReturnsParseError(t *testing.T) {
	pool, err := Open(context.Background(), "://")
	if err == nil {
		if pool != nil {
			pool.Close()
		}

		t.Fatal("Open() error = nil, want DSN parsing error")
	}

	if pool != nil {
		pool.Close()
		t.Fatal("Open() pool is not nil after error")
	}

	if !strings.Contains(err.Error(), "create postgres pool") {
		t.Fatalf("Open() error = %q, want create pool context", err)
	}
}

func TestOpenReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pool, err := Open(ctx, "postgres://user:password@localhost:5432/database?sslmode=disable")
	if err == nil {
		if pool != nil {
			pool.Close()
		}

		t.Fatal("Open() error = nil, want context cancellation error")
	}

	if pool != nil {
		pool.Close()
		t.Fatal("Open() pool is not nil after error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Open() error = %v, want context.Canceled", err)
	}
}
