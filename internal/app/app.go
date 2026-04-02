// Package app wires all components together and manages the server lifecycle.
// Dependency injection and graceful shutdown live here; no business logic.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/agabani/service-template-go/api"
	httpadapter "github.com/agabani/service-template-go/internal/adapters/http"
	"github.com/agabani/service-template-go/internal/adapters/http/handler"
	"github.com/agabani/service-template-go/internal/adapters/http/openapi"
	"github.com/agabani/service-template-go/internal/adapters/postgres"
	"github.com/agabani/service-template-go/internal/adapters/telemetry"
	"github.com/agabani/service-template-go/internal/config"
	"github.com/agabani/service-template-go/internal/domain/account"
	"github.com/agabani/service-template-go/internal/domain/user"
)

type App struct {
	server                *http.Server
	shutdown              telemetry.ShutdownFunc
	pool                  interface{ Close() }
	deregisterPoolMetrics func()
	shutdownTimeout       time.Duration
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	shutdownTelemetry, err := telemetry.Setup(ctx,
		cfg.Telemetry.ServiceName,
		cfg.Telemetry.ServiceVersion,
		cfg.Telemetry.OTLPEndpoint,
	)
	if err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	deregisterPoolMetrics, err := postgres.RegisterPoolMetrics(pool)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("register pool metrics: %w", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	accountRepo := postgres.NewAccountRepository(pool)

	userSvc := user.NewService(userRepo)
	accountSvc := account.NewService(accountRepo)

	validator, err := openapi.NewValidator(api.Spec)
	if err != nil {
		return nil, fmt.Errorf("create openapi validator: %w", err)
	}

	router := httpadapter.NewRouter(httpadapter.RouterDeps{
		HealthHandler:  handler.NewHealthHandler(pool),
		UserHandler:    handler.NewUserHandler(userSvc),
		AccountHandler: handler.NewAccountHandler(accountSvc),
		Validator:      validator,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &App{
		server:                server,
		shutdown:              shutdownTelemetry,
		pool:                  pool,
		deregisterPoolMetrics: deregisterPoolMetrics,
		shutdownTimeout:       cfg.HTTP.ShutdownTimeout,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", a.server.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	slog.Info("server started", "addr", ln.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		if err := a.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return a.gracefulShutdown()
	case err := <-errCh:
		return err
	}
}

func (a *App) gracefulShutdown() error {
	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	a.deregisterPoolMetrics()
	a.pool.Close()
	if err := a.shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("telemetry shutdown: %w", err)
	}
	slog.Info("shutdown complete")
	return nil
}
