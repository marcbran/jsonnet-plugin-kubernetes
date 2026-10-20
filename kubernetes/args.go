package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
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
