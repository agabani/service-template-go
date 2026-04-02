package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "github.com/agabani/service-template-go/internal/adapters/postgres"
	meterName  = "github.com/agabani/service-template-go/internal/adapters/postgres"
)

type contextKey int

const startTimeKey contextKey = 0

type queryStartData struct {
	startTime time.Time
	operation string
}

// queryTracer implements pgx.QueryTracer to create an OTel client span and
// record SLI metrics for every Query, QueryRow, and Exec call.
type queryTracer struct {
	tracer            trace.Tracer
	operationDuration metric.Float64Histogram
	operationTotal    metric.Int64Counter
}

func newQueryTracer() (*queryTracer, error) {
	meter := otel.Meter(meterName)

	operationDuration, err := meter.Float64Histogram(
		"db.client.operation.duration",
		metric.WithDescription("Duration of database client operations."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1),
	)
	if err != nil {
		return nil, fmt.Errorf("create db.client.operation.duration: %w", err)
	}

	operationTotal, err := meter.Int64Counter(
		"db.client.operation.total",
		metric.WithDescription("Total count of database client operations."),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create db.client.operation.total: %w", err)
	}

	return &queryTracer{
		tracer:            otel.Tracer(tracerName),
		operationDuration: operationDuration,
		operationTotal:    operationTotal,
	}, nil
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	sql := strings.TrimSpace(data.SQL)
	ctx, _ = t.tracer.Start(ctx, dbSpanName(sql),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			semconv.DBOperation(sqlVerb(sql)),
			semconv.DBStatement(sql),
		),
	)
	return context.WithValue(ctx, startTimeKey, queryStartData{
		startTime: time.Now(),
		operation: sqlVerb(sql),
	})
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
	span.End()

	qd, ok := ctx.Value(startTimeKey).(queryStartData)
	if !ok {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", qd.operation),
		attribute.Bool("error", data.Err != nil),
	)
	t.operationDuration.Record(ctx, time.Since(qd.startTime).Seconds(), attrs)
	t.operationTotal.Add(ctx, 1, attrs)
}

// dbSpanName returns a concise span name following OTel DB conventions:
// "{VERB} {table}" where the table is extractable, otherwise just "{VERB}".
func dbSpanName(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "db.query"
	}
	verb := strings.ToUpper(fields[0])
	switch verb {
	case "INSERT":
		if len(fields) >= 3 && strings.EqualFold(fields[1], "INTO") {
			return "INSERT " + fields[2]
		}
	case "UPDATE":
		if len(fields) >= 2 {
			return "UPDATE " + fields[1]
		}
	case "DELETE":
		if len(fields) >= 3 && strings.EqualFold(fields[1], "FROM") {
			return "DELETE " + fields[2]
		}
	}
	return verb
}

// sqlVerb returns the uppercase SQL command keyword (SELECT, INSERT, …).
func sqlVerb(sql string) string {
	if i := strings.IndexByte(sql, ' '); i > 0 {
		return strings.ToUpper(sql[:i])
	}
	return strings.ToUpper(sql)
}
