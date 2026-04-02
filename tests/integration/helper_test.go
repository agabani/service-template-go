//go:build integration

package integration_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/agabani/service-template-go/api"
	httpadapter "github.com/agabani/service-template-go/internal/adapters/http"
	"github.com/agabani/service-template-go/internal/adapters/http/handler"
	"github.com/agabani/service-template-go/internal/adapters/http/openapi"
	pgadapter "github.com/agabani/service-template-go/internal/adapters/postgres"
	"github.com/agabani/service-template-go/internal/config"
	"github.com/agabani/service-template-go/internal/domain/account"
	"github.com/agabani/service-template-go/internal/domain/user"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	pool, _ := setupDatabase(t)

	validator, err := openapi.NewValidator(api.Spec)
	require.NoError(t, err)

	router := httpadapter.NewRouter(httpadapter.RouterDeps{
		HealthHandler:  handler.NewHealthHandler(pool),
		UserHandler:    handler.NewUserHandler(user.NewService(pgadapter.NewUserRepository(pool))),
		AccountHandler: handler.NewAccountHandler(account.NewService(pgadapter.NewAccountRepository(pool))),
		Validator:      validator,
	})

	return httptest.NewServer(router)
}

func setupDatabase(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { pgc.Terminate(ctx) }) //nolint:errcheck

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	require.NoError(t, pgadapter.MigrateUp(dsn))

	pool, err := pgadapter.NewPool(ctx, config.DatabaseConfig{
		URL:      dsn,
		MaxConns: 5,
		MinConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool, dsn
}
