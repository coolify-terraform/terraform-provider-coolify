// Package dbtest provides shared test helpers for database resource tests.
package dbtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/acctest"
)

// MockState holds the mutable state of a mock database server.
type MockState struct {
	mu          sync.Mutex
	UUID        string
	Name        string
	Description string
	Image       string
	ExtraFields map[string]interface{}
	Deleted     bool
}

// buildResponse returns the JSON-serializable map for a GET response.
func (s *MockState) buildResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"uuid":                      s.UUID,
		"name":                      s.Name,
		"description":               s.Description,
		"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
		"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
		"environment_name":          "production",
		"image":                     s.Image,
		"is_public":                 false,
		"public_port":               nil,
		"limits_memory":             "0",
		"limits_memory_swap":        "0",
		"limits_memory_swappiness":  60,
		"limits_memory_reservation": "0",
		"limits_cpus":               "0",
		"limits_cpuset":             "0",
		"limits_cpu_shares":         1024,
		"status":                    "running",
		"internal_db_url":           "",
	}
	for k, v := range s.ExtraFields {
		resp[k] = v
	}
	return resp
}

// applyPatch updates the state from a PATCH request body.
func (s *MockState) applyPatch(body map[string]interface{}) {
	if v, ok := body["name"].(string); ok {
		s.Name = v
	}
	if v, ok := body["description"].(string); ok {
		s.Description = v
	}
	if v, ok := body["image"].(string); ok {
		s.Image = v
	}
	for k := range s.ExtraFields {
		if v, ok := body[k]; ok {
			s.ExtraFields[k] = v
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// validateCreateBody decodes the POST body and checks that all required fields
// are present. Returns true on success, or writes an error response and returns false.
func validateCreateBody(w http.ResponseWriter, r *http.Request) bool {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return false
	}
	for _, field := range []string{"project_uuid", "server_uuid", "environment_name"} {
		if _, ok := body[field]; !ok {
			http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
			return false
		}
	}
	return true
}

// NewMockServer creates an httptest.Server that simulates the Coolify database
// API for the given database type. extraFields are db-specific fields included
// in GET responses and updatable via PATCH (e.g., {"redis_password": "pass"}).
func NewMockServer(dbType, name, image string, extraFields map[string]interface{}) (*httptest.Server, *MockState) {
	// Seed common fields as defaults so applyPatch can update them.
	merged := map[string]interface{}{
		"is_log_drain_enabled":  false,
		"is_include_timestamps": false,
		"enable_ssl":            false,
		"ssl_mode":              "",
	}
	for k, v := range extraFields {
		merged[k] = v
	}
	state := &MockState{
		UUID:        "aaaa0001-0001-4000-8000-000000000001",
		Name:        name,
		Image:       image,
		ExtraFields: merged,
	}

	dbPath := fmt.Sprintf("/api/v1/databases/%s", state.UUID)
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/"+dbType:
			if !validateCreateBody(w, r) {
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{"uuid": state.UUID})

		case r.Method == http.MethodGet && r.URL.Path == dbPath:
			if state.Deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, state.buildResponse())

		case r.Method == http.MethodPatch && r.URL.Path == dbPath:
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			state.applyPatch(body)
			writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})

		case r.Method == http.MethodDelete && r.URL.Path == dbPath:
			state.Deleted = true
			writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	return srv, state
}
