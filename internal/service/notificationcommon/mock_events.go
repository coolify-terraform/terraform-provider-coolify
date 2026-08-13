package notificationcommon

import (
	"encoding/json"
	"net/http"
)

// EventStore holds the 14 shared Coolify notification event flags used by
// httptest mock servers in unit tests. Embed it in channel-specific mocks.
//
// JSON keys follow Coolify's pattern: <event>_<channel>_notifications
// (e.g. deployment_failure_discord_notifications).
type EventStore struct {
	DeploymentSuccess    bool
	DeploymentFailure    bool
	StatusChange         bool
	BackupSuccess        bool
	BackupFailure        bool
	ScheduledTaskSuccess bool
	ScheduledTaskFailure bool
	DockerCleanupSuccess bool
	DockerCleanupFailure bool
	ServerDiskUsage      bool
	ServerReachable      bool
	ServerUnreachable    bool
	ServerPatch          bool
	TraefikOutdated      bool
}

// EventJSONKey returns the Coolify PATCH/GET JSON key for a schema event attr.
// channel is the API channel slug (discord, slack, email, telegram, pushover, webhook).
func EventJSONKey(channel, eventAttr string) string {
	return eventAttr + "_" + channel + "_notifications"
}

// fieldPtrs maps EventAttributeNames entries to EventStore fields.
// Keep attrs in the same order/names as eventNames in common.go.
func (e *EventStore) fieldPtrs() []struct {
	attr string
	ptr  *bool
} {
	return []struct {
		attr string
		ptr  *bool
	}{
		{"deployment_success", &e.DeploymentSuccess},
		{"deployment_failure", &e.DeploymentFailure},
		{"status_change", &e.StatusChange},
		{"backup_success", &e.BackupSuccess},
		{"backup_failure", &e.BackupFailure},
		{"scheduled_task_success", &e.ScheduledTaskSuccess},
		{"scheduled_task_failure", &e.ScheduledTaskFailure},
		{"docker_cleanup_success", &e.DockerCleanupSuccess},
		{"docker_cleanup_failure", &e.DockerCleanupFailure},
		{"server_disk_usage", &e.ServerDiskUsage},
		{"server_reachable", &e.ServerReachable},
		{"server_unreachable", &e.ServerUnreachable},
		{"server_patch", &e.ServerPatch},
		{"traefik_outdated", &e.TraefikOutdated},
	}
}

// PutSnapshot writes all 14 event flags into out under Coolify JSON keys for channel.
func (e *EventStore) PutSnapshot(out map[string]interface{}, channel string) {
	for _, f := range e.fieldPtrs() {
		out[EventJSONKey(channel, f.attr)] = *f.ptr
	}
}

// ApplyBody sets event flags from a decoded PATCH body when keys are present.
func (e *EventStore) ApplyBody(channel string, body map[string]interface{}) {
	for _, f := range e.fieldPtrs() {
		if v, ok := body[EventJSONKey(channel, f.attr)].(bool); ok {
			*f.ptr = v
		}
	}
}

// EventAllowedFields returns a map with all 14 event JSON keys set to true for
// Coolify-style unknown-field rejection in mock PATCH handlers.
func EventAllowedFields(channel string) map[string]bool {
	out := make(map[string]bool, len(eventNames))
	for _, e := range eventNames {
		out[EventJSONKey(channel, e.attr)] = true
	}
	return out
}

// MergeAllowed returns a copy of base with each extra key set to true.
func MergeAllowed(base map[string]bool, extra ...string) map[string]bool {
	out := make(map[string]bool, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for _, k := range extra {
		out[k] = true
	}
	return out
}

// RejectUnknownFields writes a Coolify-like 422 when body contains keys not in
// allowed. Returns true if the request was rejected (caller should return).
func RejectUnknownFields(w http.ResponseWriter, body map[string]interface{}, allowed map[string]bool) bool {
	for k := range body {
		if !allowed[k] {
			http.Error(w, `{"message":"Validation failed.","errors":{"`+k+`":["This field is not allowed."]}}`, http.StatusUnprocessableEntity)
			return true
		}
	}
	return false
}

// BoolFromBody sets *dst when key is present as a JSON bool.
func BoolFromBody(body map[string]interface{}, key string, dst *bool) {
	if v, ok := body[key].(bool); ok {
		*dst = v
	}
}

// StringFromBody sets *dst when key is present as a JSON string.
func StringFromBody(body map[string]interface{}, key string, dst *string) {
	if v, ok := body[key].(string); ok {
		*dst = v
	}
}

// DecodeJSONBody decodes r.Body into a map, or writes 400 and returns nil, false.
func DecodeJSONBody(w http.ResponseWriter, r *http.Request) (map[string]interface{}, bool) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"message":"bad json"}`, http.StatusBadRequest)
		return nil, false
	}
	return body, true
}

// WriteJSON encodes v as application/json.
func WriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
