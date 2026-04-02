package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureHandler struct {
	records []slog.Record
}

func (c *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (c *captureHandler) Handle(_ context.Context, r slog.Record) error {
	c.records = append(c.records, r)
	return nil
}
func (c *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return &captureHandler{} }
func (c *captureHandler) WithGroup(_ string) slog.Handler      { return &captureHandler{} }

func TestMultiHandler_Handle_fanOut(t *testing.T) {
	a, b := &captureHandler{}, &captureHandler{}
	h := newMultiHandler(a, b)
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	require.NoError(t, h.Handle(context.Background(), rec))
	assert.Len(t, a.records, 1)
	assert.Len(t, b.records, 1)
}

func TestMultiHandler_Enabled_anyHandler(t *testing.T) {
	h := newMultiHandler(&captureHandler{})
	assert.True(t, h.Enabled(context.Background(), slog.LevelInfo))
}

func TestMultiHandler_WithAttrs_propagates(t *testing.T) {
	h := newMultiHandler(&captureHandler{})
	assert.NotNil(t, h.WithAttrs([]slog.Attr{slog.String("k", "v")}))
}

func TestMultiHandler_WithGroup_propagates(t *testing.T) {
	h := newMultiHandler(&captureHandler{})
	assert.NotNil(t, h.WithGroup("g"))
}
