// Package handler provides HTTP request handlers for the REST API.
// Each handler decodes the request, delegates to a domain service, and encodes
// the response. Business logic must not live here — it belongs in the domain.
// This package must not import the postgres adapter.
package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
	"github.com/agabani/service-template-go/internal/domain"
)

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", jsonapi.ContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(r.Context(), "encode response", "error", err)
	}
}

func writeBadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	writeJSON(w, r, http.StatusBadRequest, jsonapi.ErrorDocument{
		Errors: []jsonapi.Error{{Status: "400", Title: "Bad Request", Detail: detail}},
	})
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, apiErr := mapDomainError(err)
	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "internal error", "error", err)
	}
	writeJSON(w, r, status, jsonapi.ErrorDocument{Errors: []jsonapi.Error{apiErr}})
}

// decodeBody decodes a JSON:API request document and returns the resource data.
// It writes a 400 response and returns false on any decode or structural error.
func decodeBody[T any](w http.ResponseWriter, r *http.Request) (*jsonapi.Resource[T], bool) {
	var doc jsonapi.Document[T]
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		writeBadRequest(w, r, err.Error())
		return nil, false
	}
	if doc.Data == nil {
		writeBadRequest(w, r, "data is required")
		return nil, false
	}
	return doc.Data, true
}

// pathUUID parses a UUID from a named path segment.
// It writes a 400 response and returns false if the value is absent or malformed.
func pathUUID(w http.ResponseWriter, r *http.Request, key, detail string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(key))
	if err != nil {
		writeBadRequest(w, r, detail)
		return uuid.Nil, false
	}
	return id, true
}

// parsePage extracts page[size], page[after], and page[before] query parameters.
// page[after] and page[before] are mutually exclusive.
// It writes a 400 response and returns false on any parse error.
func parsePage(w http.ResponseWriter, r *http.Request) (domain.PageInput, bool) {
	input := domain.PageInput{Size: domain.PageSizeDefault}
	q := r.URL.Query()

	if s := q.Get("page[size]"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			writeBadRequest(w, r, "page[size] must be a positive integer")
			return domain.PageInput{}, false
		}
		input.Size = n
	}

	hasAfter := q.Has("page[after]")
	hasBefore := q.Has("page[before]")

	if hasAfter && hasBefore {
		writeBadRequest(w, r, "page[after] and page[before] are mutually exclusive")
		return domain.PageInput{}, false
	}

	if hasAfter {
		cursor, ok := decodeCursor(q.Get("page[after]"))
		if !ok {
			writeBadRequest(w, r, "page[after] is invalid")
			return domain.PageInput{}, false
		}
		input.After = cursor
	}

	if hasBefore {
		cursor, ok := decodeCursor(q.Get("page[before]"))
		if !ok {
			writeBadRequest(w, r, "page[before] is invalid")
			return domain.PageInput{}, false
		}
		input.Before = cursor
	}

	return input, true
}

// paginationURL builds an absolute URL for a pagination link.
// It sets the given param to the encoded cursor and removes the opposite direction param.
func paginationURL(r *http.Request, param string, cursor *domain.PageCursor) *string {
	if cursor == nil {
		return nil
	}
	u := *r.URL
	if u.Host == "" {
		u.Host = r.Host
	}
	if u.Scheme == "" {
		if r.TLS != nil {
			u.Scheme = "https"
		} else {
			u.Scheme = "http"
		}
	}
	q := u.Query()
	q.Del("page[after]")
	q.Del("page[before]")
	q.Set(param, encodeCursor(cursor))
	u.RawQuery = q.Encode()
	s := u.String()
	return &s
}

type cursorPayload struct {
	ID int64 `json:"i"`
}

func encodeCursor(c *domain.PageCursor) string {
	b, _ := json.Marshal(cursorPayload{ID: c.ID})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (*domain.PageCursor, bool) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, false
	}
	var p cursorPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, false
	}
	if p.ID <= 0 {
		return nil, false
	}
	return &domain.PageCursor{ID: p.ID}, true
}

func mapDomainError(err error) (int, jsonapi.Error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, jsonapi.Error{
			Status: "404",
			Title:  "Not Found",
			Detail: err.Error(),
		}
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, jsonapi.Error{
			Status: "409",
			Title:  "Conflict",
			Detail: err.Error(),
		}
	case errors.Is(err, domain.ErrValidation):
		return http.StatusUnprocessableEntity, jsonapi.Error{
			Status: "422",
			Title:  "Unprocessable Entity",
			Detail: err.Error(),
		}
	default:
		return http.StatusInternalServerError, jsonapi.Error{
			Status: "500",
			Title:  "Internal Server Error",
		}
	}
}
