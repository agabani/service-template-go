// Package http is the primary HTTP adapter. It assembles the request router,
// applies middleware, and wires handlers to routes. New routes belong here;
// business logic belongs in the domain layer, not in handlers.
package http

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/agabani/service-template-go/internal/adapters/http/middleware"
)

type healthHandler interface {
	Live(http.ResponseWriter, *http.Request)
	Ready(http.ResponseWriter, *http.Request)
}

type userHandler interface {
	List(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
}

type accountHandler interface {
	ListByUser(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
}

type validatorMiddleware interface {
	Middleware(next http.Handler) http.Handler
}

type RouterDeps struct {
	HealthHandler  healthHandler
	UserHandler    userHandler
	AccountHandler accountHandler
	Validator      validatorMiddleware
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", spanName("GET /health/live", deps.HealthHandler.Live))
	mux.HandleFunc("GET /health/ready", spanName("GET /health/ready", deps.HealthHandler.Ready))

	mux.HandleFunc("GET /users", spanName("GET /users", deps.UserHandler.List))
	mux.HandleFunc("POST /users", spanName("POST /users", deps.UserHandler.Create))
	mux.HandleFunc("GET /users/{id}", spanName("GET /users/{id}", deps.UserHandler.Get))
	mux.HandleFunc("PATCH /users/{id}", spanName("PATCH /users/{id}", deps.UserHandler.Update))
	mux.HandleFunc("DELETE /users/{id}", spanName("DELETE /users/{id}", deps.UserHandler.Delete))

	mux.HandleFunc("GET /users/{userID}/accounts", spanName("GET /users/{userID}/accounts", deps.AccountHandler.ListByUser))
	mux.HandleFunc("POST /users/{userID}/accounts", spanName("POST /users/{userID}/accounts", deps.AccountHandler.Create))

	mux.HandleFunc("GET /accounts/{id}", spanName("GET /accounts/{id}", deps.AccountHandler.Get))
	mux.HandleFunc("PATCH /accounts/{id}", spanName("PATCH /accounts/{id}", deps.AccountHandler.Update))
	mux.HandleFunc("DELETE /accounts/{id}", spanName("DELETE /accounts/{id}", deps.AccountHandler.Delete))

	var h http.Handler = mux
	h = deps.Validator.Middleware(h)
	h = middleware.Logger(h)
	h = middleware.RequestID(h)
	h = middleware.Recovery(h)
	h = otelhttp.NewHandler(h, "server")

	return h
}

// spanName wraps a handler to update the active OTel span name to the matched
// route pattern and tag the otelhttp labeler so http.route is included in
// the http.server.request.duration metric recorded by the outer otelhttp handler.
func spanName(pattern string, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trace.SpanFromContext(r.Context()).SetName(pattern)
		l, _ := otelhttp.LabelerFromContext(r.Context())
		l.Add(attribute.String("http.route", pattern))
		fn(w, r)
	}
}
