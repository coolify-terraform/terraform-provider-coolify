package client

import "encoding/json"

// jsonObjectEmpty reports whether v encodes as an empty JSON object.
// Coolify's validateIncomingRequest rejects that body with 400 Empty JSON.
func jsonObjectEmpty(v any) bool {
	if v == nil {
		return true
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return false
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	return len(body) == 0
}
