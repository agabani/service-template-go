package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type ShutdownFunc func(context.Context) error

func Setup(ctx context.Context, serviceName, serviceVersion, otlpEndpoint string) (ShutdownFunc, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if otlpEndpoint == "" {
		otel.SetTracerProvider(tracenoop.NewTracerProvider())
		otel.SetMeterProvider(metricnoop.NewMeterProvider())
		global.SetLoggerProvider(lognoop.NewLoggerProvider())
		return func(_ context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	// shutdowns accumulates provider teardown funcs in creation order.
	// Any failure below calls shutdownAll to clean up already-started providers.
	var shutdowns []func(context.Context) error
	shutdownAll := func(ctx context.Context) {
		for i := len(shutdowns) - 1; i >= 0; i-- {
			_ = shutdowns[i](ctx)
		}
	}

	logExporter, err := newLogExporter(ctx, otlpEndpoint)
	if err != nil {
		shutdownAll(ctx)
		return nil, fmt.Errorf("create otlp log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)
	global.SetLoggerProvider(lp)
	shutdowns = append(shutdowns, lp.Shutdown)

	metricExporter, err := newMetricExporter(ctx, otlpEndpoint)
	if err != nil {
		shutdownAll(ctx)
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	traceExporter, err := newTraceExporter(ctx, otlpEndpoint)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)),
	)
	otel.SetTracerProvider(tp)
	shutdowns = append(shutdowns, tp.Shutdown)

	return func(ctx context.Context) error {
		var errs []error
		for i := len(shutdowns) - 1; i >= 0; i-- {
			if err := shutdowns[i](ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, nil
}

// newLogExporter selects gRPC or HTTP based on the URL scheme:
//
//	grpc://host:port   — gRPC, insecure
//	grpcs://host:port  — gRPC, TLS
//	http://host:port   — HTTP, insecure
//	https://host:port  — HTTP, TLS
func newLogExporter(ctx context.Context, rawEndpoint string) (sdklog.Exporter, error) {
	u, err := url.Parse(rawEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", rawEndpoint, err)
	}

	switch u.Scheme {
	case "grpc":
		return otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(u.Host),
			otlploggrpc.WithTLSCredentials(insecure.NewCredentials()),
		)
	case "grpcs":
		return otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(u.Host),
			otlploggrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		)
	case "http":
		return otlploghttp.New(ctx,
			otlploghttp.WithEndpoint(u.Host),
			otlploghttp.WithInsecure(),
		)
	case "https":
		return otlploghttp.New(ctx,
			otlploghttp.WithEndpoint(u.Host),
		)
	default:
		return nil, fmt.Errorf("unsupported OTLP endpoint scheme %q (use http, https, grpc, or grpcs)", u.Scheme)
	}
}

// newMetricExporter selects gRPC or HTTP based on the URL scheme:
//
//	grpc://host:port   — gRPC, insecure
//	grpcs://host:port  — gRPC, TLS
//	http://host:port   — HTTP, insecure
//	https://host:port  — HTTP, TLS
func newMetricExporter(ctx context.Context, rawEndpoint string) (sdkmetric.Exporter, error) {
	u, err := url.Parse(rawEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", rawEndpoint, err)
	}

	switch u.Scheme {
	case "grpc":
		return otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(u.Host),
			otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()),
		)
	case "grpcs":
		return otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(u.Host),
			otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		)
	case "http":
		return otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(u.Host),
			otlpmetrichttp.WithInsecure(),
		)
	case "https":
		return otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(u.Host),
		)
	default:
		return nil, fmt.Errorf("unsupported OTLP endpoint scheme %q (use http, https, grpc, or grpcs)", u.Scheme)
	}
}

// newTraceExporter selects gRPC or HTTP based on the URL scheme:
//
//	grpc://host:port   — gRPC, insecure
//	grpcs://host:port  — gRPC, TLS
//	http://host:port   — HTTP, insecure
//	https://host:port  — HTTP, TLS
func newTraceExporter(ctx context.Context, rawEndpoint string) (sdktrace.SpanExporter, error) {
	u, err := url.Parse(rawEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", rawEndpoint, err)
	}

	switch u.Scheme {
	case "grpc":
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(u.Host),
			otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
		)
	case "grpcs":
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(u.Host),
			otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		)
	case "http":
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(u.Host),
			otlptracehttp.WithInsecure(),
		)
	case "https":
		return otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(u.Host),
		)
	default:
		return nil, fmt.Errorf("unsupported OTLP endpoint scheme %q (use http, https, grpc, or grpcs)", u.Scheme)
	}
}
