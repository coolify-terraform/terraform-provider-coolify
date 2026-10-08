package acctest

import (
	"encoding/json"
	"fmt"
	"io"
)

// DecodeObject reads a JSON object and keeps raw values so a missing key
// can be told apart from a present one. Mocks must not substitute a
// server default and then pretend the provider sent that key.
func DecodeObject(r io.Reader) (map[string]json.RawMessage, error) {
	var body map[string]json.RawMessage
	dec := json.NewDecoder(r)
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}
	if body == nil {
		body = map[string]json.RawMessage{}
	}
	return body, nil
}

// DecodeOptional unmarshals key when it is present and not null.
// A missing or null key returns nil and increments absent when absent is non-nil.
func DecodeOptional[T any](body map[string]json.RawMessage, key string, absent *int) (*T, error) {
	raw, ok := body[key]
	if !ok || string(raw) == "null" {
		if absent != nil {
			*absent++
		}
		return nil, nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode %s: %w", key, err)
	}
	return &value, nil
}
