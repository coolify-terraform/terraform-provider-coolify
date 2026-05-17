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
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// maxResponseSize limits API response bodies to 10 MB to prevent OOM
// from a malicious or compromised server.
const maxResponseSize = 10 << 20

// Client is the Coolify API client.
type Client struct {
	BaseURL    string
	apiToken   string // unexported: prevents %+v leaking the token
	HTTPClient *http.Client
	UserAgent  string
	listCache  listCache
}

// listCache is a short-lived, thread-safe cache for GET list responses.
// It prevents redundant API calls when multiple resources with the same
// parent are read during a single plan/apply cycle.
type listCache struct {
	mu      sync.Mutex
	entries map[string]listCacheEntry
}

type listCacheEntry struct {
	data    []byte
	expires time.Time
}

const listCacheTTL = 5 * time.Second

// getCached returns cached response bytes for the given path, or nil if
// the cache is empty or expired.
func (lc *listCache) get(path string) []byte {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.entries == nil {
		return nil
	}
	e, ok := lc.entries[path]
	if !ok || time.Now().After(e.expires) {
		delete(lc.entries, path)
		return nil
	}
	return e.data
}

// set stores response bytes in the cache with a TTL.
func (lc *listCache) set(path string, data []byte) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.entries == nil {
		lc.entries = make(map[string]listCacheEntry)
	}
	lc.entries[path] = listCacheEntry{data: data, expires: time.Now().Add(listCacheTTL)}
}

// invalidate removes a cache entry (called after mutating operations).
func (lc *listCache) invalidate(path string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	delete(lc.entries, path)
}

// RetryConfig holds user-configurable retry settings.
type RetryConfig struct {
	Attempts int
	MinWait  time.Duration
	MaxWait  time.Duration
}

// New creates a new Coolify API client.
func New(baseURL, apiToken string, opts ...RetryConfig) *Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	if len(opts) > 0 && opts[0].Attempts > 0 {
		rc.RetryMax = opts[0].Attempts
	}
	if len(opts) > 0 && opts[0].MinWait > 0 {
		rc.RetryWaitMin = opts[0].MinWait
	}
	if len(opts) > 0 && opts[0].MaxWait > 0 {
		rc.RetryWaitMax = opts[0].MaxWait
	}
	rc.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}
		if resp.StatusCode >= 500 {
			switch resp.Request.Method {
			case http.MethodGet, http.MethodPatch:
				return true, nil
			default:
				return false, nil
			}
		}
		return false, nil
	}
	rc.Logger = nil
	httpClient := rc.StandardClient()
	httpClient.Timeout = 30 * time.Second

	return &Client{
		BaseURL:    baseURL,
		apiToken:   apiToken,
		HTTPClient: httpClient,
		UserAgent:  "terraform-provider-coolify",
	}
}

// GetVersion returns the Coolify instance version string.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	return c.doText(ctx, "/api/v1/version")
}

// GetHealth returns the Coolify instance health status string.
func (c *Client) GetHealth(ctx context.Context) (string, error) {
	return c.doText(ctx, "/api/v1/health")
}

// doText performs a GET request and returns the response body as a trimmed
// string. Handles both plain text and JSON-encoded string responses.
func (c *Client) doText(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request for %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", fmt.Errorf("reading response for %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("api error for %s (status %d): %s", path, resp.StatusCode, extractAPIMessage(body))
	}

	var unquoted string
	if json.Unmarshal(body, &unquoted) == nil {
		return unquoted, nil
	}
	return strings.TrimSpace(string(body)), nil
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

// doCachedList performs a GET request with short-lived caching. Repeated
// calls with the same path within the TTL window return cached data
// without hitting the API. Use for List endpoints where multiple
// Terraform resources share the same parent.
func (c *Client) doCachedList(ctx context.Context, path string, result interface{}) error {
	if cached := c.listCache.get(path); cached != nil {
		return json.Unmarshal(cached, result)
	}
	// Make the real API call and capture raw bytes.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Message: fmt.Sprintf("resource not found: %s", extractAPIMessage(respBody))}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, extractAPIMessage(respBody))
	}

	c.listCache.set(path, respBody)
	if result != nil {
		return json.Unmarshal(respBody, result)
	}
	return nil
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
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	// Check 404 first, regardless of expectedStatus.
	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Message: fmt.Sprintf("resource not found: %s", extractAPIMessage(respBody))}
	}
	if expectedStatus != 0 && resp.StatusCode != expectedStatus {
		return fmt.Errorf("expected status %d, got %d: %s", expectedStatus, resp.StatusCode, extractAPIMessage(respBody))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, extractAPIMessage(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// validParentTypes is the set of allowed parent resource types for compound
// API paths like /api/v1/{parentType}/{parentUUID}/scheduled-tasks.
var validParentTypes = map[string]bool{
	"applications": true,
	"services":     true,
	"databases":    true,
}

func validateParentType(pt string) error {
	if !validParentTypes[pt] {
		return fmt.Errorf("invalid parent type %q: must be one of applications, services, databases", pt)
	}
	return nil
}

// extractAPIMessage attempts to parse a JSON error response from the Coolify
// API and return the human-readable "message" field. Falls back to the raw
// body if parsing fails or no message field is present.
func extractAPIMessage(body []byte) string {
	var parsed struct {
		Message string                     `json:"message"`
		Errors  map[string]json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.Message != "" {
		if len(parsed.Errors) > 0 {
			parts := make([]string, 0, len(parsed.Errors))
			for field, detail := range parsed.Errors {
				parts = append(parts, field+": "+string(detail))
			}
			return parsed.Message + " " + strings.Join(parts, "; ")
		}
		return parsed.Message
	}
	s := strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' {
			return -1
		}
		return r
	}, string(body))
	if len(s) > 200 {
		s = s[:200] + "... (truncated)"
	}
	return "[raw API response] " + s
}

// RetryDelete retries a delete operation with backoff when the error is
// retryable (e.g., resource still has dependents). It returns nil on
// success or NotFound, or the last error after exhausting retries.
func RetryDelete(ctx context.Context, attempts int, delay time.Duration, deleteFn func() error, isRetryable func(error) bool) error {
	for range attempts {
		err := deleteFn()
		if err == nil || IsNotFound(err) {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return deleteFn()
}
