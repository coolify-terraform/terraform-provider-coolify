package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Client is the Coolify API client.
type Client struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

// New creates a new Coolify API client.
func New(baseURL, apiToken string) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIToken:   apiToken,
		HTTPClient: &http.Client{},
	}
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if expectedStatus != 0 {
		if resp.StatusCode != expectedStatus {
			return fmt.Errorf("expected status %d, got %d: %s", expectedStatus, resp.StatusCode, string(respBody))
		}
	} else {
		if resp.StatusCode == http.StatusNotFound {
			return &NotFoundError{Message: fmt.Sprintf("resource not found: %s", string(respBody))}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// -------------------------------------------------------------------------
// MongoDB legacy helpers (used by the mongodb resource)
// -------------------------------------------------------------------------

// CreateDatabaseRequest is a freeform map used by the MongoDB legacy API.
type CreateDatabaseRequest map[string]interface{}

// UpdateDatabaseRequest is a freeform map used by the MongoDB legacy API.
type UpdateDatabaseRequest map[string]interface{}

// DatabaseResponse is the shape returned by the legacy database endpoints.
type DatabaseResponse struct {
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	Description             string `json:"description,omitempty"`
	Image                   string `json:"image,omitempty"`
	IsPublic                bool   `json:"is_public"`
	PublicPort              *int64 `json:"public_port,omitempty"`
	ProjectUUID             string `json:"project_uuid,omitempty"`
	ServerUUID              string `json:"server_uuid,omitempty"`
	EnvironmentName         string `json:"environment_name,omitempty"`
	MongoInitdbRootUsername  string `json:"mongo_initdb_root_username,omitempty"`
	MongoInitdbRootPassword string `json:"mongo_initdb_root_password,omitempty"`
	MongoInitdbDatabase     string `json:"mongo_initdb_database,omitempty"`
}

func (c *Client) CreateMongodbDatabaseLegacy(body CreateDatabaseRequest) (*DatabaseResponse, error) {
	var resp DatabaseResponse
	if err := c.doWithStatus(context.Background(), http.MethodPost, "/api/v1/databases/mongodb", body, &resp, http.StatusCreated); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetDatabaseLegacy(uuid string) (*DatabaseResponse, error) {
	var resp DatabaseResponse
	if err := c.do(context.Background(), http.MethodGet, fmt.Sprintf("/api/v1/databases/%s", uuid), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpdateDatabaseLegacy(uuid string, body UpdateDatabaseRequest) error {
	return c.do(context.Background(), http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s", uuid), body, nil)
}

func (c *Client) DeleteDatabaseLegacy(uuid string) error {
	return c.do(context.Background(), http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s", uuid), nil, nil)
}
