//go:build integration

package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
)

const repositoryIntegrationTestTimeout = 30 * time.Second

func TestIntegration_UserRepositoryCreate(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), repositoryIntegrationTestTimeout)
	defer cancel()

	pool := openMigratedTestDatabase(t, ctx, dsn)
	repository := postgres.NewUserRepository(pool)
	passwordHash := []byte("opaque-password-hash")

	user, err := repository.Create(ctx, "alice", passwordHash)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.ID <= 0 {
		t.Errorf("Create() user ID = %d, want positive value", user.ID)
	}
	if user.Login != "alice" {
		t.Errorf("Create() login = %q, want %q", user.Login, "alice")
	}
	if user.CreatedAt.IsZero() {
		t.Error("Create() created_at is zero")
	}

	var storedPasswordHash []byte
	if err := pool.QueryRow(
		ctx,
		"SELECT password_hash FROM gopherkeeper.users WHERE id = $1",
		user.ID,
	).Scan(&storedPasswordHash); err != nil {
		t.Fatalf("read stored password hash: %v", err)
	}

	if !bytes.Equal(storedPasswordHash, passwordHash) {
		t.Error("stored password hash differs from repository input")
	}

	_, err = repository.Create(ctx, "alice", []byte("another-password-hash"))
	if !errors.Is(err, model.ErrLoginAlreadyExists) {
		t.Fatalf("duplicate Create() error = %v, want ErrLoginAlreadyExists", err)
	}

	var userCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM gopherkeeper.users WHERE login = $1",
		"alice",
	).Scan(&userCount); err != nil {
		t.Fatalf("count stored users: %v", err)
	}

	if userCount != 1 {
		t.Errorf("stored user count = %d, want 1", userCount)
	}
}

func openMigratedTestDatabase(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()

	adminPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	if err := adminPool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	databaseName := fmt.Sprintf("gopherkeeper_repository_test_%d", time.Now().UnixNano())
	quotedDatabaseName := pgx.Identifier{databaseName}.Sanitize()

	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+quotedDatabaseName); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	var testPool *pgxpool.Pool
	t.Cleanup(func() {
		if testPool != nil {
			testPool.Close()
		}

		dropCtx, dropCancel := context.WithTimeout(context.Background(), repositoryIntegrationTestTimeout)
		defer dropCancel()

		if _, err := adminPool.Exec(dropCtx, "DROP DATABASE "+quotedDatabaseName+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_DSN: %v", err)
	}
	cfg.ConnConfig.Database = databaseName

	testPool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}

	if err := testPool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	if err := migration.Run(testPool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return testPool
}
