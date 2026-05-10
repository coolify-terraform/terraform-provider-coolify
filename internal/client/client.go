package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// Client is the Coolify API client.
type Client struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
	UserAgent  string
}

// New creates a new Coolify API client.
func New(baseURL, apiToken string) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return true, nil
		}
		return false, nil
	}
	rc.Logger = nil
	httpClient := rc.StandardClient()
	httpClient.Timeout = 30 * time.Second

	return &Client{
		BaseURL:    baseURL,
		APIToken:   apiToken,
		HTTPClient: httpClient,
		UserAgent:  "terraform-provider-coolify",
	}
}

// GetVersion returns the Coolify instance version string.
// The version endpoint returns text/html (not JSON), so we read the
// body as a plain string instead of using the JSON-decoding do() method.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/version", nil)
	if err != nil {
		return "", fmt.Errorf("creating version request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing version request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading version response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("version API error (status %d): %s", resp.StatusCode, string(body))
	}

	v := strings.TrimSpace(string(body))
	// Handle both plain text ("v4.0.0") and JSON-encoded string ("\"v4.0.0\"")
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	return v, nil
}

// GetHealth returns the Coolify instance health status string.
// The health endpoint may return text/html (not JSON), so we read the
// body as a plain string similar to GetVersion.
func (c *Client) GetHealth(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/health", nil)
	if err != nil {
		return "", fmt.Errorf("creating health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing health request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading health response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("health API error (status %d): %s", resp.StatusCode, string(body))
	}

	h := strings.TrimSpace(string(body))
	if len(h) >= 2 && h[0] == '"' && h[len(h)-1] == '"' {
		h = h[1 : len(h)-1]
	}
	return h, nil
}

// EnableAPI enables the Coolify API.
func (c *Client) EnableAPI(ctx context.Context) error {
	if err := c.do(ctx, http.MethodGet, "/api/v1/enable", nil, nil); err != nil {
		return fmt.Errorf("enabling API: %w", err)
	}
	return nil
}

// DisableAPI disables the Coolify API.
func (c *Client) DisableAPI(ctx context.Context) error {
	if err := c.do(ctx, http.MethodGet, "/api/v1/disable", nil, nil); err != nil {
		return fmt.Errorf("disabling API: %w", err)
	}
	return nil
}

// NotFoundError is returned when the API responds with 404.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

// IsNotFound reports whether err is a NotFoundError.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}

// do executes an API request, accepting any 2xx status.
func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return c.doWithStatus(ctx, method, path, body, result, 0)
}

// doWithStatus executes an API request. When expectedStatus is non-zero only
// that exact status code is accepted; otherwise any 2xx is accepted.
func (c *Client) doWithStatus(ctx context.Context, method, path string, body interface{}, result interface{}, expectedStatus int) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	// Check 404 first, regardless of expectedStatus.
	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Message: fmt.Sprintf("resource not found: %s", string(respBody))}
	}
	if expectedStatus != 0 && resp.StatusCode != expectedStatus {
		return fmt.Errorf("expected status %d, got %d: %s", expectedStatus, resp.StatusCode, string(respBody))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
