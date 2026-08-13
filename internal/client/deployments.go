package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type Deployment struct {
	UUID       string `json:"deployment_uuid,omitempty"`
	ID         int    `json:"id,omitempty"`
	Status     string `json:"status,omitempty"`
	ServerUUID string `json:"server_uuid,omitempty"`
	// Logs is the Coolify ApplicationDeploymentQueue log payload. The API
	// returns it only when the token can read sensitive fields. The wire
	// form is either a JSON array of entries or a JSON string containing
	// that array.
	Logs json.RawMessage `json:"logs,omitempty"`
}

type deploymentLogEntry struct {
	Output string `json:"output"`
	Hidden bool   `json:"hidden"`
}

// FormatLogs returns the last maxLines visible log outputs, joined by
// newlines. Empty when logs are missing, hidden, or unreadable.
func (d Deployment) FormatLogs(maxLines int) string {
	raw := bytes.TrimSpace(d.Logs)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		raw = []byte(asString)
	}
	var entries []deploymentLogEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		s := string(raw)
		if len(s) > 2000 {
			s = s[len(s)-2000:]
		}
		return s
	}
	var lines []string
	for _, e := range entries {
		if e.Hidden || e.Output == "" {
			continue
		}
		lines = append(lines, e.Output)
	}
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

type DeployByTagInput struct {
	ForceRebuild bool `json:"force_rebuild"`
}

func (c *Client) ListDeployments(ctx context.Context) ([]Deployment, error) {
	// Coolify bug: sortBy('id') without values() produces a JSON object
	// with non-sequential keys instead of an array when deployments have
	// gaps in their indices. Try array first, fall back to object.
	// See: https://github.com/coollabsio/coolify/issues/10077
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/api/v1/deployments", nil, &raw); err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}
	var r []Deployment
	if err := json.Unmarshal(raw, &r); err == nil {
		return r, nil
	}
	var m map[string]Deployment
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("listing deployments: decoding response: %w", err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Preserve the sparse array order encoded in the object keys.
	sort.Slice(keys, func(i, j int) bool {
		ki, errI := strconv.Atoi(keys[i])
		kj, errJ := strconv.Atoi(keys[j])
		switch {
		case errI == nil && errJ == nil:
			return ki < kj
		case errI == nil:
			return true
		case errJ == nil:
			return false
		default:
			return keys[i] < keys[j]
		}
	})
	r = make([]Deployment, 0, len(keys))
	for _, k := range keys {
		r = append(r, m[k])
	}
	return r, nil
}
func (c *Client) GetDeployment(ctx context.Context, uuid string) (*Deployment, error) {
	var r Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/%s", url.PathEscape(uuid)), nil, &r); err != nil {
		return nil, fmt.Errorf("getting deployment %s: %w", uuid, err)
	}
	return &r, nil
}
func (c *Client) ListApplicationDeployments(ctx context.Context, appUUID string) ([]Deployment, error) {
	var r []Deployment
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/deployments/applications/%s", url.PathEscape(appUUID)), nil, &r); err != nil {
		return nil, fmt.Errorf("listing deployments for application %s: %w", appUUID, err)
	}
	return r, nil
}
func (c *Client) CancelDeployment(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/deployments/%s/cancel", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("cancelling deployment %s: %w", uuid, err)
	}
	return nil
}

// Deploy triggers a generic deploy (webhook-style).
func (c *Client) Deploy(ctx context.Context) error {
	if err := c.do(ctx, http.MethodPost, "/api/v1/deploy", nil, nil); err != nil {
		return fmt.Errorf("triggering deploy: %w", err)
	}
	return nil
}

func (c *Client) DeployByTag(ctx context.Context, tag string, input DeployByTagInput) error {
	q := url.Values{}
	q.Set("tag", tag)
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/deploy?%s", q.Encode()), input, nil); err != nil {
		return fmt.Errorf("deploying by tag %s: %w", tag, err)
	}
	return nil
}
