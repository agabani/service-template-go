// Package config parses environment variables into typed configuration structs.
// It has no dependencies on adapter or domain packages.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP      HTTPConfig
	Database  DatabaseConfig
	Telemetry TelemetryConfig
}

type HTTPConfig struct {
	Host            string
	Port            int
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
}

type TelemetryConfig struct {
	OTLPEndpoint   string
	ServiceName    string
	ServiceVersion string
}

func Load() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Host:            envString("HTTP_HOST", ""),
			Port:            envInt("HTTP_PORT", 8080),
			ShutdownTimeout: envDuration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
			ReadTimeout:     envDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    envDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     envDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			URL:             envString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/app?sslmode=disable"),
			MaxConns:        int32(envInt("DATABASE_MAX_CONNS", 10)),
			MinConns:        int32(envInt("DATABASE_MIN_CONNS", 2)),
			ConnMaxLifetime: envDuration("DATABASE_CONN_MAX_LIFETIME", time.Hour),
		},
		Telemetry: TelemetryConfig{
			OTLPEndpoint:   envString("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			ServiceName:    envString("OTEL_SERVICE_NAME", "service-template-go"),
			ServiceVersion: envString("OTEL_SERVICE_VERSION", "dev"),
		},
	}
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		slog.Warn("invalid env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return i
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return d
}
