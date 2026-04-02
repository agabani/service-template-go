package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(r.Context(), "panic recovered", "panic", rec)
				w.Header().Set("Content-Type", jsonapi.ContentType)
				w.WriteHeader(http.StatusInternalServerError)
				if err := json.NewEncoder(w).Encode(jsonapi.ErrorDocument{
					Errors: []jsonapi.Error{{Status: "500", Title: "Internal Server Error"}},
				}); err != nil {
					slog.ErrorContext(r.Context(), "encode recovery response", "error", err)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
