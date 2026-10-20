package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

func decodeArg[T any](arg any, name string) (T, error) {
	var out T
	raw, err := json.Marshal(arg)
	if err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

func encodeOutput(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func actionArgs(data map[string]any, keys ...string) ([]any, error) {
	for key := range data {
		if !slices.Contains(keys, key) {
			return nil, fmt.Errorf("unknown field %q", key)
		}
	}
	args := make([]any, len(keys))
	for i, key := range keys {
		value, ok := data[key]
		if !ok {
			return nil, fmt.Errorf("%s is required", key)
		}
		args[i] = value
	}
	return args, nil
}

func encodeActionOutput(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
