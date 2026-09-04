package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	// CoolifyVersion is the connected instance's version, read once during
	// provider Configure. Empty when unknown (unit tests, or a Configure that
	// never ran); callers must treat empty as "assume the newest behaviour"
	// rather than gating features off.
	CoolifyVersion    string
	cfAccessClientID  string // Cloudflare Access CF-Access-Client-Id header
	cfAccessClientSec string // Cloudflare Access CF-Access-Client-Secret header
	listCache         listCache
}

// listCache is a short-lived, thread-safe cache for GET list responses.
// It prevents redundant API calls when multiple resources with the same
// parent are read during a single plan/apply cycle.
type listCache struct {
	mu      sync.Mutex
	entries map[string]listCacheEntry
	gens    map[string]uint64
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

// generation returns the current generation for path. set only stores
// when the observed generation still matches.
func (lc *listCache) generation(path string) uint64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.gens == nil {
		return 0
	}
	return lc.gens[path]
}

// set stores response bytes in the cache with a TTL. If gen does not
// match the current generation, the write is ignored so a stale in-flight
// GET cannot refill the cache after invalidate.
func (lc *listCache) set(path string, data []byte, gen uint64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.gens != nil && lc.gens[path] != gen {
		return
	}
	if lc.entries == nil {
		lc.entries = make(map[string]listCacheEntry)
	}
	lc.entries[path] = listCacheEntry{data: data, expires: time.Now().Add(listCacheTTL)}
}

// invalidate removes a cache entry (called after mutating operations)
// and bumps the path generation so in-flight GETs cannot set stale data.
func (lc *listCache) invalidate(path string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.gens == nil {
		lc.gens = make(map[string]uint64)
	}
	lc.gens[path]++
	delete(lc.entries, path)
}

// RetryConfig holds user-configurable retry and TLS settings.
type RetryConfig struct {
	Attempts          int
	MinWait           time.Duration
	MaxWait           time.Duration
	CACert            string // PEM-encoded CA certificate to trust
	Insecure          bool   // Skip TLS certificate verification
	CFAccessClientID  string // Cloudflare Access client ID header
	CFAccessClientSec string // Cloudflare Access client secret header
}

// New creates a new Coolify API client.
func New(baseURL, apiToken string, opts ...RetryConfig) *Client {
	var cfg RetryConfig
	if len(opts) > 0 {
		cfg = opts[0]
	}
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	if cfg.Attempts > 0 {
		rc.RetryMax = cfg.Attempts
	}
	if cfg.MinWait > 0 {
		rc.RetryWaitMin = cfg.MinWait
	}
	if cfg.MaxWait > 0 {
		rc.RetryWaitMax = cfg.MaxWait
	}
	rc.CheckRetry = shouldRetry
	rc.Logger = retryablehttp.LeveledLogger(&retryLogger{})

	// Configure custom TLS before StandardClient() so the retry transport
	// wraps the correct underlying transport. Setting httpClient.Transport
	// after StandardClient() would overwrite the retry layer.
	if cfg.CACert != "" || cfg.Insecure {
		tlsCfg := &tls.Config{InsecureSkipVerify: cfg.Insecure} //nolint:gosec // user-opted insecure
		if cfg.CACert != "" {
			pool, err := x509.SystemCertPool()
			if err != nil {
				pool = x509.NewCertPool()
			}
			pool.AppendCertsFromPEM([]byte(cfg.CACert))
			tlsCfg.RootCAs = pool
		}
		rc.HTTPClient.Transport = &http.Transport{
			TLSClientConfig: tlsCfg,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1, // match go-cleanhttp default
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	// Set timeout on the inner HTTP client so it applies per-attempt, not
	// across the entire retry chain. The outer operation-level context
	// (e.g., resource Create timeouts) provides the overall ceiling.
	rc.HTTPClient.Timeout = 30 * time.Second
	httpClient := rc.StandardClient()

	return &Client{
		BaseURL:           baseURL,
		apiToken:          apiToken,
		HTTPClient:        httpClient,
		UserAgent:         "terraform-provider-coolify",
		cfAccessClientID:  cfg.CFAccessClientID,
		cfAccessClientSec: cfg.CFAccessClientSec,
	}
}

// setCommonHeaders sets Authorization, User-Agent, and optional Cloudflare Access headers.
func (c *Client) setCommonHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", c.UserAgent)
	if c.cfAccessClientID != "" {
		req.Header.Set("CF-Access-Client-Id", c.cfAccessClientID)
	}
	if c.cfAccessClientSec != "" {
		req.Header.Set("CF-Access-Client-Secret", c.cfAccessClientSec)
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
	ctx = withRequestMethod(ctx, http.MethodGet)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %s: %w", path, err)
	}
	c.setCommonHeaders(req)
	tflog.Trace(ctx, "API request", map[string]interface{}{
		"method": http.MethodGet, "path": path,
	})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing request for %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return "", fmt.Errorf("reading response for %s: %w", path, err)
	}
	tflog.Trace(ctx, "API response", map[string]interface{}{
		"path": path, "status": resp.StatusCode,
	})
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
	if err := c.do(ctx, http.MethodPost, "/api/v1/enable", nil, nil); err != nil {
		return fmt.Errorf("enabling API: %w", err)
	}
	return nil
}

// DisableAPI disables the Coolify API.
func (c *Client) DisableAPI(ctx context.Context) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/disable", nil, nil); err != nil {
		return fmt.Errorf("disabling API: %w", err)
	}
	return nil
}

// EnableMCP enables the Coolify MCP server. Requires a root team (team 0) token.
func (c *Client) EnableMCP(ctx context.Context) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/mcp/enable", nil, nil); err != nil {
		return fmt.Errorf("enabling MCP server: %w", err)
	}
	return nil
}

// DisableMCP disables the Coolify MCP server. Requires a root team (team 0) token.
func (c *Client) DisableMCP(ctx context.Context) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/mcp/disable", nil, nil); err != nil {
		return fmt.Errorf("disabling MCP server: %w", err)
	}
	return nil
}

// requestMethodKey stores the HTTP method on the request context so
// CheckRetry can classify transport errors when resp is nil (timeouts).
type requestMethodKey struct{}

func withRequestMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, requestMethodKey{}, method)
}

func requestMethodFromContext(ctx context.Context, resp *http.Response) string {
	if m, ok := ctx.Value(requestMethodKey{}).(string); ok && m != "" {
		return m
	}
	if resp != nil && resp.Request != nil {
		return resp.Request.Method
	}
	return ""
}

// shouldRetry is the retryablehttp CheckRetry policy.
// 429 retries every method. Transport errors and 5xx retry GET and PATCH
// only; POST/PUT/DELETE must not retry on timeout (duplicate creates).
func shouldRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if err != nil {
		switch requestMethodFromContext(ctx, resp) {
		case http.MethodGet, http.MethodPatch:
			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		default:
			return false, nil
		}
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
		tflog.Trace(ctx, "API cache hit", map[string]interface{}{"path": path})
		if err := json.Unmarshal(cached, result); err == nil {
			return nil
		}
		// Evict corrupted or type-mismatched entry and fall through to fresh fetch.
		c.listCache.invalidate(path)
	}
	// Snapshot generation after the miss (and any corrupt-entry invalidate)
	// so a later Create/Update invalidate makes this GET's set a no-op.
	gen := c.listCache.generation(path)
	// Make the real API call and capture raw bytes.
	tflog.Trace(ctx, "API request", map[string]interface{}{"method": "GET", "path": path})
	ctx = withRequestMethod(ctx, http.MethodGet)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request for GET %s: %w", path, err)
	}
	c.setCommonHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request for GET %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	tflog.Trace(ctx, "API response", map[string]interface{}{
		"path": path, "status": resp.StatusCode,
		"body": redactJSON(respBody),
	})

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Message: fmt.Sprintf("resource not found (GET %s): %s", path, extractAPIMessage(respBody))}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned status %d for GET %s: %s", resp.StatusCode, path, extractAPIMessage(respBody))
	}

	// Cache only after successful unmarshal to avoid storing malformed data.
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	c.listCache.set(path, respBody, gen)
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
		tflog.Trace(ctx, "API request", map[string]interface{}{
			"method": method, "path": path,
			"body": redactJSON(data),
		})
	} else {
		tflog.Trace(ctx, "API request", map[string]interface{}{
			"method": method, "path": path,
		})
	}

	ctx = withRequestMethod(ctx, method)
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request for %s %s: %w", method, path, err)
	}
	c.setCommonHeaders(req)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request for %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	tflog.Trace(ctx, "API response", map[string]interface{}{
		"method": method, "path": path,
		"status":       resp.StatusCode,
		"body_excerpt": redactJSON(respBody),
	})

	// Check 404 first, regardless of expectedStatus.
	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{Message: fmt.Sprintf("resource not found (%s %s): %s", method, path, extractAPIMessage(respBody))}
	}
	if expectedStatus != 0 && resp.StatusCode != expectedStatus {
		return fmt.Errorf("expected status %d, got %d for %s %s: %s", expectedStatus, resp.StatusCode, method, path, extractAPIMessage(respBody))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api error (status %d) for %s %s: %s", resp.StatusCode, method, path, extractAPIMessage(respBody))
	}

	if result != nil {
		if len(respBody) == 0 {
			return fmt.Errorf("API returned status %d with empty body for %s %s (expected JSON)", resp.StatusCode, method, path)
		} else if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// sensitiveKeys are JSON field names whose values are redacted in logs.
var sensitiveKeys = map[string]bool{
	"password": true, "private_key": true, "token": true,
	"secret": true, "client_secret": true, "webhook_secret": true,
	"redis_password": true, "postgres_password": true, "mysql_password": true,
	"mysql_root_password": true, "mariadb_password": true,
	"mariadb_root_password": true, "mongo_initdb_root_password": true,
	"clickhouse_admin_password": true, "dragonfly_password": true,
	"keydb_password": true, "http_basic_auth_password": true,
	"value":              true, // env var payloads use {"key":"DB_PASS","value":"secret"}
	"docker_compose_raw": true, "docker_compose": true,
	"cloud_init_script": true, "dockerfile": true,
	"script":      true, // cloud-init YAML bodies
	"webhook_url": true,
}

// redactJSON replaces sensitive field values with [REDACTED] in a JSON byte
// slice for safe logging. Handles objects, arrays, and nested structures.
// Returns the original string (truncated) if unmarshaling fails.
func redactJSON(data []byte) string {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "[non-JSON body omitted]"
	}
	redactValue(raw)
	out, err := json.Marshal(raw)
	if err != nil {
		return truncateString(string(data), 500)
	}
	return truncateString(string(out), 500)
}

func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	if sensitiveKeys[lower] {
		return true
	}
	return strings.Contains(lower, "password") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "private_key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "api_key") ||
		strings.Contains(lower, "license_key") ||
		strings.Contains(lower, "user_key")
}

// mapHasKeyAndSecret reports whether m looks like an S3 credential object
// (both "key" and "secret", case-insensitive). Lone "key" fields are env-var
// names and must not be redacted.
func mapHasKeyAndSecret(m map[string]interface{}) bool {
	hasKey, hasSecret := false, false
	for k := range m {
		switch strings.ToLower(k) {
		case "key":
			hasKey = true
		case "secret":
			hasSecret = true
		}
	}
	return hasKey && hasSecret
}

func redactValue(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		redactAccessKey := mapHasKeyAndSecret(val)
		for k, child := range val {
			if isSensitiveField(k) || (redactAccessKey && strings.EqualFold(k, "key")) {
				val[k] = "[REDACTED]"
			} else {
				redactValue(child)
			}
		}
	case []interface{}:
		for _, item := range val {
			redactValue(item)
		}
	}
}

// truncateString truncates s to maxLen runes, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
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
// API and return the human-readable "message" field. Raw response bodies are
// never appended (they may contain secrets or HTML). When an "errors" map is
// present, values for sensitive field names are redacted.
func extractAPIMessage(body []byte) string {
	var parsed struct {
		Message string                     `json:"message"`
		Errors  map[string]json.RawMessage `json:"errors"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.Message != "" {
		if len(parsed.Errors) > 0 {
			parts := make([]string, 0, len(parsed.Errors))
			for field, detail := range parsed.Errors {
				if isSensitiveField(field) {
					parts = append(parts, field+": [REDACTED]")
				} else {
					parts = append(parts, field+": "+string(detail))
				}
			}
			return parsed.Message + " " + strings.Join(parts, "; ")
		}
		return parsed.Message
	}
	return "API error response omitted"
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
	finalErr := deleteFn()
	if finalErr == nil || IsNotFound(finalErr) {
		return nil
	}
	return finalErr
}

// retryLogger adapts retryablehttp's LeveledLogger to tflog at TRACE level.
// Since retryablehttp doesn't pass context, we log to the background context.
type retryLogger struct{}

func (l *retryLogger) Error(msg string, keysAndValues ...interface{}) {
	tflog.Trace(context.Background(), "[retry] "+msg, toMap(keysAndValues))
}
func (l *retryLogger) Warn(msg string, keysAndValues ...interface{}) {
	tflog.Trace(context.Background(), "[retry] "+msg, toMap(keysAndValues))
}
func (l *retryLogger) Info(msg string, keysAndValues ...interface{}) {
	tflog.Trace(context.Background(), "[retry] "+msg, toMap(keysAndValues))
}
func (l *retryLogger) Debug(msg string, keysAndValues ...interface{}) {
	tflog.Trace(context.Background(), "[retry] "+msg, toMap(keysAndValues))
}

func toMap(keysAndValues []interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if k, ok := keysAndValues[i].(string); ok {
			m[k] = keysAndValues[i+1]
		}
	}
	return m
}

// PollUntilDeleted polls a get function with exponential backoff (500ms to 5s).
// It returns gone=true when the resource is confirmed NotFound. A poll-context
// deadline with the resource still present returns gone=false and that
// deadline error. The first GET runs immediately; later probes wait. getFn
// receives pollCtx so the deadline cancels in-flight GETs.
func PollUntilDeleted(ctx context.Context, getFn func(context.Context) error) (bool, error) {
	pollCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
	}

	delay := 500 * time.Millisecond
	const maxDelay = 5 * time.Second
	for {
		err := getFn(pollCtx)
		if IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		select {
		case <-pollCtx.Done():
			return false, pollCtx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
