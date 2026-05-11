package spectest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	validator "github.com/pb33f/libopenapi-validator"
)

// WithSpecValidation wraps an http.Handler to validate all requests and
// responses against the OpenAPI spec. Violations are logged as test errors.
//
// Usage:
//
//	srv := httptest.NewServer(spectest.WithSpecValidation(t, "coolify-v4",
//	    acctest.WithVersionEndpoint(mux),
//	))
func WithSpecValidation(t testing.TB, specVersion string, next http.Handler) http.Handler {
	return withSpecValidation(t, specVersion, next, true)
}

// WithSpecAudit is like WithSpecValidation but logs violations as warnings
// instead of failing the test. Use this when the spec itself has quality
// issues (e.g., wrong response types) that would cause false failures.
func WithSpecAudit(t testing.TB, specVersion string, next http.Handler) http.Handler {
	return withSpecValidation(t, specVersion, next, false)
}

func withSpecValidation(t testing.TB, specVersion string, next http.Handler, strict bool) http.Handler {
	v, err := loadValidator(specVersion)
	if err != nil {
		t.Fatalf("failed to load validator for spec %s: %v", specVersion, err)
	}
	return &validatingHandler{
		inner:     next,
		validator: v,
		t:         t,
		strict:    strict,
		skipPaths: []string{
			"/api/v1/version", // returns text/html, validator expects JSON
		},
	}
}

type validatingHandler struct {
	inner     http.Handler
	validator validator.Validator
	t         testing.TB
	strict    bool // true = t.Errorf (fail test), false = t.Logf (audit only)
	skipPaths []string
}

func (vh *validatingHandler) report(format string, args ...interface{}) {
	vh.t.Helper()
	if vh.strict {
		vh.t.Errorf(format, args...)
	} else {
		vh.t.Logf(format, args...)
	}
}

func (vh *validatingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	apiPath := r.URL.Path
	if idx := strings.Index(apiPath, "/api/v1/"); idx >= 0 {
		apiPath = apiPath[idx:]
	}

	for _, prefix := range vh.skipPaths {
		if strings.HasPrefix(apiPath, prefix) {
			vh.inner.ServeHTTP(w, r)
			return
		}
	}

	// Build a request with the spec's base URL so the validator can match paths.
	specReq := r.Clone(r.Context())
	specReq.URL.Scheme = "https"
	specReq.URL.Host = "app.coolify.io"
	specReq.URL.Path = apiPath
	specReq.RequestURI = ""

	// Buffer request body for validation.
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		specReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Validate request.
	valid, validationErrs := vh.validator.ValidateHttpRequest(specReq)
	if !valid {
		for _, e := range validationErrs {
			vh.report("[OpenAPI] request %s %s violates spec: %s", r.Method, apiPath, e.Message)
			for _, se := range e.SchemaValidationErrors {
				vh.report("[OpenAPI]   schema: %s", se.Reason)
			}
		}
	}

	rec := httptest.NewRecorder()
	vh.inner.ServeHTTP(rec, r)

	for k, vs := range rec.Header() {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(rec.Code)
	respBody := rec.Body.Bytes()
	_, _ = w.Write(respBody)

	// Validate response.
	resp := &http.Response{
		StatusCode: rec.Code,
		Header:     rec.Header(),
		Body:       io.NopCloser(bytes.NewReader(respBody)),
	}
	valid, validationErrs = vh.validator.ValidateHttpResponse(specReq, resp)
	if !valid {
		for _, e := range validationErrs {
			vh.report("[OpenAPI] response %s %s (status %d) violates spec: %s",
				r.Method, apiPath, rec.Code, e.Message)
			for _, se := range e.SchemaValidationErrors {
				vh.report("[OpenAPI]   schema: %s", se.Reason)
			}
		}
	}
}
