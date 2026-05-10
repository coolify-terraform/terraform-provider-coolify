package spectest

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
)

// ValidatingTransport wraps an http.RoundTripper and validates every
// request and response against an OpenAPI spec. Use it to catch
// mismatches between the client code and the real API contract.
type ValidatingTransport struct {
	Inner     http.RoundTripper
	Validator validator.Validator
	T         testing.TB

	// SkipPaths contains path prefixes to skip validation for (e.g.,
	// endpoints not in the spec like /api/v1/storages).
	SkipPaths []string
}

// NewValidatingTransport creates a ValidatingTransport that checks all
// HTTP traffic against the given OpenAPI spec.
func NewValidatingTransport(t testing.TB, inner http.RoundTripper, doc *libopenapi.Document) *ValidatingTransport {
	v, errs := validator.NewValidator(*doc)
	if len(errs) > 0 {
		t.Fatalf("failed to create OpenAPI validator: %v", errs)
	}
	return &ValidatingTransport{
		Inner:     inner,
		Validator: v,
		T:         t,
	}
}

func (vt *ValidatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Strip the host prefix to get the API path for validation.
	apiPath := req.URL.Path
	if idx := strings.Index(apiPath, "/api/v1/"); idx >= 0 {
		apiPath = apiPath[idx:]
	}

	// Check if this path should skip validation.
	for _, prefix := range vt.SkipPaths {
		if strings.HasPrefix(apiPath, prefix) {
			return vt.Inner.RoundTrip(req)
		}
	}

	// Buffer the request body so we can read it twice (validation + actual send).
	var reqBodyBytes []byte
	if req.Body != nil {
		var err error
		reqBodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	}

	// Validate the request against the spec.
	valid, validationErrs := vt.Validator.ValidateHttpRequest(req)
	if !valid {
		for _, e := range validationErrs {
			vt.T.Errorf("[OpenAPI] request %s %s violates spec: %s", req.Method, apiPath, e.Message)
			for _, se := range e.SchemaValidationErrors {
				vt.T.Errorf("[OpenAPI]   schema: %s", se.Reason)
			}
		}
	}

	// Restore the request body for the actual transport.
	if reqBodyBytes != nil {
		req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	}

	// Forward to the inner transport.
	resp, err := vt.Inner.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Buffer the response body so we can validate and still return it.
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))

	// Validate the response against the spec.
	valid, validationErrs = vt.Validator.ValidateHttpResponse(req, resp)
	if !valid {
		for _, e := range validationErrs {
			vt.T.Errorf("[OpenAPI] response %s %s (status %d) violates spec: %s",
				req.Method, apiPath, resp.StatusCode, e.Message)
			for _, se := range e.SchemaValidationErrors {
				vt.T.Errorf("[OpenAPI]   schema: %s", se.Reason)
			}
		}
	}

	// Restore the response body.
	resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))
	return resp, nil
}
