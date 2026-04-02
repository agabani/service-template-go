package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agabani/service-template-go/internal/adapters/http/middleware"
)

// captureHandler is a slog.Handler that records the last handled record.
type captureHandler struct{ last slog.Record }

func (h *captureHandler) Enabled(context.Context, slog.Level) bool      { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error { h.last = r; return nil }
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler            { return h }
func (h *captureHandler) WithGroup(string) slog.Handler                 { return h }

func TestRequestID_generatesIDWhenAbsent(t *testing.T) {
	var capturedID string
	handler := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedID = middleware.RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.NotEmpty(t, capturedID)
	assert.Equal(t, capturedID, rec.Header().Get(middleware.RequestIDHeader))
}

func TestRequestIDHandler_injectsRequestIDIntoRecord(t *testing.T) {
	const id = "test-id"
	cap := &captureHandler{}
	logger := slog.New(middleware.NewRequestIDHandler(cap))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, id)
	var enrichedCtx context.Context
	middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		enrichedCtx = r.Context()
	})).ServeHTTP(httptest.NewRecorder(), req)

	logger.InfoContext(enrichedCtx, "msg")

	var found string
	cap.last.Attrs(func(a slog.Attr) bool {
		if a.Key == "request_id" {
			found = a.Value.String()
		}
		return true
	})
	require.Equal(t, id, found)
}

func TestRequestIDHandler_omitsAttributeWhenNoID(t *testing.T) {
	cap := &captureHandler{}
	logger := slog.New(middleware.NewRequestIDHandler(cap))

	logger.InfoContext(context.Background(), "msg")

	var found bool
	cap.last.Attrs(func(a slog.Attr) bool {
		if a.Key == "request_id" {
			found = true
		}
		return true
	})
	assert.False(t, found)
}

func TestRequestID_propagatesIncomingID(t *testing.T) {
	const incomingID = "test-request-id"

	var capturedID string
	handler := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedID = middleware.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, incomingID)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, incomingID, capturedID)
	assert.Equal(t, incomingID, rec.Header().Get(middleware.RequestIDHeader))
}
