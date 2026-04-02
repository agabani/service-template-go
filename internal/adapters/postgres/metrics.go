package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// RegisterPoolMetrics registers observable gauges that report connection pool
// health on every metric collection cycle. The returned function unregisters
// the gauges and should be called before the pool is closed.
func RegisterPoolMetrics(pool *pgxpool.Pool) (func(), error) {
	meter := otel.Meter(meterName)

	usedConns, err := meter.Int64ObservableGauge(
		"db.client.connection.pool.used",
		metric.WithDescription("Number of connections currently acquired from the pool."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create db.client.connection.pool.used: %w", err)
	}

	idleConns, err := meter.Int64ObservableGauge(
		"db.client.connection.pool.idle",
		metric.WithDescription("Number of idle connections in the pool."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create db.client.connection.pool.idle: %w", err)
	}

	maxConns, err := meter.Int64ObservableGauge(
		"db.client.connection.pool.max",
		metric.WithDescription("Maximum number of open connections allowed in the pool."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create db.client.connection.pool.max: %w", err)
	}

	reg, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		stat := pool.Stat()
		o.ObserveInt64(usedConns, int64(stat.AcquiredConns()))
		o.ObserveInt64(idleConns, int64(stat.IdleConns()))
		o.ObserveInt64(maxConns, int64(stat.MaxConns()))
		return nil
	}, usedConns, idleConns, maxConns)
	if err != nil {
		return nil, fmt.Errorf("register pool metrics callback: %w", err)
	}

	return func() { _ = reg.Unregister() }, nil
}
