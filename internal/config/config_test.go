package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/agabani/service-template-go/internal/config"
)

func TestLoad_returnsDefaults(t *testing.T) {
	cfg := config.Load()

	assert.Equal(t, 8080, cfg.HTTP.Port)
	assert.Equal(t, 30*time.Second, cfg.HTTP.ShutdownTimeout)
	assert.Equal(t, 5*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTP.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.HTTP.IdleTimeout)
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/app?sslmode=disable", cfg.Database.URL)
	assert.Equal(t, int32(10), cfg.Database.MaxConns)
	assert.Equal(t, int32(2), cfg.Database.MinConns)
	assert.Equal(t, time.Hour, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, "service-template-go", cfg.Telemetry.ServiceName)
	assert.Equal(t, "dev", cfg.Telemetry.ServiceVersion)
}

func TestLoad_readsEnvVars(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("OTEL_SERVICE_NAME", "my-service")

	cfg := config.Load()

	assert.Equal(t, 9090, cfg.HTTP.Port)
	assert.Equal(t, "my-service", cfg.Telemetry.ServiceName)
}

func TestLoad_invalidEnvVar_usesDefault(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-number")

	cfg := config.Load()

	assert.Equal(t, 8080, cfg.HTTP.Port)
}
