package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetup_noOp(t *testing.T) {
	shutdown, err := Setup(context.Background(), "svc", "1.0.0", "")
	require.NoError(t, err)
	require.NoError(t, shutdown(context.Background()))
}

func TestSetup_unsupportedScheme(t *testing.T) {
	_, err := Setup(context.Background(), "svc", "1.0.0", "ftp://host:4317")
	require.Error(t, err)
}

func TestNewLogExporter_unsupportedScheme(t *testing.T) {
	_, err := newLogExporter(context.Background(), "ftp://host:4317")
	require.Error(t, err)
}

func TestNewMetricExporter_unsupportedScheme(t *testing.T) {
	_, err := newMetricExporter(context.Background(), "ftp://host:4317")
	require.Error(t, err)
}

func TestNewTraceExporter_unsupportedScheme(t *testing.T) {
	_, err := newTraceExporter(context.Background(), "ftp://host:4317")
	require.Error(t, err)
}
