package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/bridges/otelslog"

	"github.com/agabani/service-template-go/internal/adapters/http/middleware"
	"github.com/agabani/service-template-go/internal/adapters/postgres"
	"github.com/agabani/service-template-go/internal/app"
	"github.com/agabani/service-template-go/internal/config"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	godotenv.Load() //nolint:errcheck

	slog.SetDefault(slog.New(middleware.NewRequestIDHandler(newMultiHandler(
		slog.NewTextHandler(os.Stderr, nil),
		otelslog.NewHandler("service-template-go"),
	))))

	root := &cobra.Command{
		Use:     "server",
		Short:   "Service template HTTP server",
		Version: version,
	}

	root.AddCommand(serveCmd(), migrateCmd(), versionCmd())

	if err := root.Execute(); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func serveCmd() *cobra.Command {
	var (
		host      string
		port      int
		dbURL     string
		otlpEndpt string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := config.Load()

			if cmd.Flags().Changed("host") {
				cfg.HTTP.Host = host
			}
			if cmd.Flags().Changed("port") {
				cfg.HTTP.Port = port
			}
			if cmd.Flags().Changed("database-url") {
				cfg.Database.URL = dbURL
			}
			if cmd.Flags().Changed("otlp-endpoint") {
				cfg.Telemetry.OTLPEndpoint = otlpEndpt
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			a, err := app.New(ctx, cfg)
			if err != nil {
				return fmt.Errorf("initialise app: %w", err)
			}

			return a.Run(ctx)
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "HTTP host to bind (overrides HTTP_HOST)")
	cmd.Flags().IntVar(&port, "port", 0, "HTTP port to bind (overrides HTTP_PORT)")
	cmd.Flags().StringVar(&dbURL, "database-url", "", "PostgreSQL connection URL (overrides DATABASE_URL)")
	cmd.Flags().StringVar(&otlpEndpt, "otlp-endpoint", "", "OTLP exporter endpoint (overrides OTEL_EXPORTER_OTLP_ENDPOINT)")

	return cmd
}

func migrateCmd() *cobra.Command {
	var dbURL string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
	}

	up := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			url := resolveDBURL(cmd, dbURL)
			if err := postgres.MigrateUp(url); err != nil {
				return fmt.Errorf("migrate up: %w", err)
			}
			slog.Info("migrations applied")
			return nil
		},
	}

	down := &cobra.Command{
		Use:   "down",
		Short: "Revert all migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			url := resolveDBURL(cmd, dbURL)
			if err := postgres.MigrateDown(url); err != nil {
				return fmt.Errorf("migrate down: %w", err)
			}
			slog.Info("migrations reverted")
			return nil
		},
	}

	for _, sub := range []*cobra.Command{up, down} {
		sub.Flags().StringVar(&dbURL, "database-url", "", "PostgreSQL connection URL (overrides DATABASE_URL)")
		cmd.AddCommand(sub)
	}

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version)
		},
	}
}

func resolveDBURL(cmd *cobra.Command, flagValue string) string {
	if cmd.Flags().Changed("database-url") {
		return flagValue
	}
	return config.Load().Database.URL
}
