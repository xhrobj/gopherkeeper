//go:build integration

package migration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
)

const integrationTestTimeout = 30 * time.Second

func TestRunAppliesEmbeddedMigrationsOnce(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	if err := adminPool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	databaseName := fmt.Sprintf("gopherkeeper_test_%d", time.Now().UnixNano())
	quotedDatabaseName := pgx.Identifier{databaseName}.Sanitize()

	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+quotedDatabaseName); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	var testPool *pgxpool.Pool
	t.Cleanup(func() {
		if testPool != nil {
			testPool.Close()
		}

		dropCtx, dropCancel := context.WithTimeout(context.Background(), integrationTestTimeout)
		defer dropCancel()

		if _, err := adminPool.Exec(dropCtx, "DROP DATABASE "+quotedDatabaseName+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})

	testPool = openTestPool(t, ctx, dsn, databaseName)

	if err := migration.Run(testPool); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	if err := migration.Run(testPool); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	assertMigrationState(t, ctx, testPool)
}

func openTestPool(t *testing.T, ctx context.Context, dsn, databaseName string) *pgxpool.Pool {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_DSN: %v", err)
	}
	cfg.ConnConfig.Database = databaseName

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test database: %v", err)
	}

	return pool
}

func assertMigrationState(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	var schemaExists bool
	if err := pool.QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'gopherkeeper')",
	).Scan(&schemaExists); err != nil {
		t.Fatalf("check gopherkeeper schema: %v", err)
	}
	if !schemaExists {
		t.Fatal("gopherkeeper schema does not exist")
	}

	var version int64
	var dirty bool
	if err := pool.QueryRow(
		ctx,
		"SELECT version, dirty FROM public.schema_migrations",
	).Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration state: %v", err)
	}

	if version != 1 {
		t.Errorf("migration version = %d, want 1", version)
	}
	if dirty {
		t.Error("migration state is dirty")
	}

	var migrationTableCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM pg_tables WHERE tablename = 'schema_migrations'",
	).Scan(&migrationTableCount); err != nil {
		t.Fatalf("count migration tables: %v", err)
	}

	if migrationTableCount != 1 {
		t.Errorf("schema_migrations table count = %d, want 1", migrationTableCount)
	}
}
