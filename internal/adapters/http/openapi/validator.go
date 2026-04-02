package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/agabani/service-template-go/internal/adapters/http/jsonapi"
)

type Validator struct {
	router routers.Router
}

func NewValidator(specData []byte) (*Validator, error) {
	// Register the JSON:API content type as a JSON body so kin-openapi can
	// decode and validate request bodies with that media type.
	openapi3filter.RegisterBodyDecoder("application/vnd.api+json", func(body io.Reader, _ http.Header, _ *openapi3.SchemaRef, _ openapi3filter.EncodingFn) (any, error) {
		var v any
		if err := json.NewDecoder(body).Decode(&v); err != nil {
			return nil, &openapi3filter.ParseError{Kind: openapi3filter.KindInvalidFormat, Cause: err}
		}
		return v, nil
	})

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(specData)
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate openapi spec: %w", err)
	}
	doc.Servers = nil

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("create openapi router: %w", err)
	}
	return &Validator{router: router}, nil
}

func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, pathParams, err := v.router.FindRoute(r)
		if err != nil {
			if !errors.Is(err, routers.ErrPathNotFound) && !errors.Is(err, routers.ErrMethodNotAllowed) {
				slog.WarnContext(r.Context(), "openapi route lookup failed", "error", err)
			}
			next.ServeHTTP(w, r)
			return
		}

		input := &openapi3filter.RequestValidationInput{
			Request:    r,
			PathParams: pathParams,
			Route:      route,
			Options: &openapi3filter.Options{
				AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			},
		}

		if err := openapi3filter.ValidateRequest(r.Context(), input); err != nil {
			w.Header().Set("Content-Type", jsonapi.ContentType)
			w.WriteHeader(http.StatusBadRequest)
			if encErr := json.NewEncoder(w).Encode(jsonapi.ErrorDocument{
				Errors: validationErrors(err),
			}); encErr != nil {
				slog.ErrorContext(r.Context(), "encode validation error response", "error", encErr)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validationErrors(err error) []jsonapi.Error {
	var multiErr openapi3.MultiError
	if errors.As(err, &multiErr) {
		out := make([]jsonapi.Error, 0, len(multiErr))
		for _, e := range multiErr {
			out = append(out, validationErrors(e)...)
		}
		if len(out) == 0 {
			return []jsonapi.Error{{Status: "400", Title: "Bad Request"}}
		}
		return out
	}

	var schemaErr *openapi3.SchemaError
	if errors.As(err, &schemaErr) {
		return []jsonapi.Error{schemaErrorToAPIError(schemaErr)}
	}

	var reqErr *openapi3filter.RequestError
	if errors.As(err, &reqErr) && reqErr.Parameter != nil {
		return []jsonapi.Error{parameterErrorToAPIError(reqErr)}
	}

	return []jsonapi.Error{{
		Status: "400",
		Title:  "Bad Request",
		Detail: firstLine(err.Error()),
	}}
}

func schemaErrorToAPIError(err *openapi3.SchemaError) jsonapi.Error {
	parts := err.JSONPointer()
	pointer := ""
	if len(parts) > 0 {
		pointer = "/" + strings.Join(parts, "/")
	}
	return jsonapi.Error{
		Status: "400",
		Title:  "Bad Request",
		Detail: buildSchemaDetail(err),
		Source: &jsonapi.ErrorSource{Pointer: pointer},
	}
}

func buildSchemaDetail(err *openapi3.SchemaError) string {
	detail := err.Reason

	if err.Schema == nil || err.SchemaField == "" {
		return detail
	}

	schema := err.Schema

	switch err.SchemaField {
	case "required":
		// reason is already self-descriptive, e.g. "property \"email\" is missing"
	case "format":
		detail = fmt.Sprintf("%s (expected format: %s)", detail, schema.Format)
	case "enum":
		vals := make([]string, 0, len(schema.Enum))
		for _, v := range schema.Enum {
			vals = append(vals, fmt.Sprintf("%v", v))
		}
		detail = fmt.Sprintf("%s (allowed: %s)", detail, strings.Join(vals, ", "))
	case "minLength":
		detail = fmt.Sprintf("%s (minimum length: %d)", detail, schema.MinLength)
	case "maxLength":
		if schema.MaxLength != nil {
			detail = fmt.Sprintf("%s (maximum length: %d)", detail, *schema.MaxLength)
		}
	case "minimum":
		if schema.Min != nil {
			detail = fmt.Sprintf("%s (minimum: %v)", detail, *schema.Min)
		}
	case "maximum":
		if schema.Max != nil {
			detail = fmt.Sprintf("%s (maximum: %v)", detail, *schema.Max)
		}
	case "type":
		detail = fmt.Sprintf("%s (expected type: %s)", detail, schema.Type)
	default:
		detail = fmt.Sprintf("%s (%s)", detail, err.SchemaField)
	}

	return detail
}

func parameterErrorToAPIError(err *openapi3filter.RequestError) jsonapi.Error {
	in := err.Parameter.In
	name := err.Parameter.Name
	detail := firstLine(err.Err.Error())

	source := &jsonapi.ErrorSource{}
	switch in {
	case "path":
		source.Pointer = "/data/" + name
	case "query", "header":
		source.Parameter = name
	}

	return jsonapi.Error{
		Status: "400",
		Title:  "Bad Request",
		Detail: fmt.Sprintf("parameter %q: %s", name, detail),
		Source: source,
	}
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
