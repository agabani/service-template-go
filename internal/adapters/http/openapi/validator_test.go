package openapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
)

const minimalSpec = `
openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths:
  /items:
    post:
      requestBody:
        required: true
        content:
          application/vnd.api+json:
            schema:
              type: object
              required: [data]
              properties:
                data:
                  type: object
      responses:
        "201":
          description: Created
`

func TestFirstLine_noNewline(t *testing.T) {
	assert.Equal(t, "hello", firstLine("hello"))
}

func TestFirstLine_withNewline(t *testing.T) {
	assert.Equal(t, "first", firstLine("first\nsecond\nthird"))
}

func TestValidationErrors_plainError(t *testing.T) {
	errs := validationErrors(errors.New("oops\nmore detail"))
	require.Len(t, errs, 1)
	assert.Equal(t, "400", errs[0].Status)
	assert.Equal(t, "oops", errs[0].Detail)
}

func TestValidationErrors_multiError(t *testing.T) {
	multi := openapi3.MultiError{
		&openapi3.SchemaError{Reason: "err1"},
		errors.New("err2"),
	}
	errs := validationErrors(multi)
	assert.Len(t, errs, 2)
}

func TestValidationErrors_schemaError(t *testing.T) {
	se := &openapi3.SchemaError{
		Reason:      "value too short",
		Schema:      &openapi3.Schema{MinLength: 5},
		SchemaField: "minLength",
	}
	errs := validationErrors(se)
	require.Len(t, errs, 1)
	assert.Equal(t, "400", errs[0].Status)
	assert.Contains(t, errs[0].Detail, "minimum length: 5")
	assert.NotNil(t, errs[0].Source)
}

func TestValidationErrors_requestErrorWithParameter(t *testing.T) {
	re := &openapi3filter.RequestError{
		Parameter: &openapi3.Parameter{In: "query", Name: "page"},
		Err:       errors.New("must be integer"),
	}
	errs := validationErrors(re)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Detail, `"page"`)
	assert.Equal(t, "page", errs[0].Source.Parameter)
}

func TestBuildSchemaDetail(t *testing.T) {
	tests := []struct {
		name        string
		schemaField string
		schema      *openapi3.Schema
		reason      string
		wantContain string
	}{
		{"minLength", "minLength", &openapi3.Schema{MinLength: 3}, "too short", "minimum length: 3"},
		{"maxLength", "maxLength", &openapi3.Schema{MaxLength: ptr(uint64(10))}, "too long", "maximum length: 10"},
		{"enum", "enum", &openapi3.Schema{Enum: []any{"a", "b"}}, "invalid", "allowed: a, b"},
		{"format", "format", &openapi3.Schema{Format: "uuid"}, "bad format", "expected format: uuid"},
		{"default", "pattern", &openapi3.Schema{}, "mismatch", "pattern"},
		{"nil schema", "", nil, "bare reason", "bare reason"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			se := &openapi3.SchemaError{Reason: tc.reason, Schema: tc.schema, SchemaField: tc.schemaField}
			assert.Contains(t, buildSchemaDetail(se), tc.wantContain)
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestValidatorMiddleware_unknownRoute_passesThrough(t *testing.T) {
	v, err := NewValidator([]byte(minimalSpec))
	require.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := v.Middleware(next)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestValidatorMiddleware_validRequest_passesThrough(t *testing.T) {
	v, err := NewValidator([]byte(minimalSpec))
	require.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := v.Middleware(next)

	body := strings.NewReader(`{"data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/items", body)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestValidatorMiddleware_invalidRequest_returns400(t *testing.T) {
	v, err := NewValidator([]byte(minimalSpec))
	require.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := v.Middleware(next)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/items", body)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, jsonapi.ContentType, rec.Header().Get("Content-Type"))

	var resp jsonapi.ErrorDocument
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Errors)
}
