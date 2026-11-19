package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFetchInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []any
		want      FetchInput
		wantError string
	}{
		{
			name: "without fields",
			args: []any{"test-context", "/api/v1/namespaces/default/pods", []any{}},
			want: FetchInput{Context: "test-context", Path: "/api/v1/namespaces/default/pods", Fields: []string{}},
		},
		{
			name: "with fields",
			args: []any{"test-context", "/api/v1/pods", []any{"metadata.name", "status.phase"}},
			want: FetchInput{Context: "test-context", Path: "/api/v1/pods", Fields: []string{"metadata.name", "status.phase"}},
		},
		{
			name:      "wrong arg count",
			args:      []any{"test-context", "/api/v1/pods"},
			wantError: "expected ctx, path, fields, and optional options",
		},
		{
			name: "with selectors",
			args: []any{"test-context", "/api/v1/pods", []any{}, map[string]any{"labelSelector": "app=api", "fieldSelector": "status.phase=Running"}},
			want: FetchInput{Context: "test-context", Path: "/api/v1/pods", Fields: []string{}, Selectors: Selectors{Label: "app=api", Field: "status.phase=Running"}},
		},
		{
			name: "null options and selectors",
			args: []any{"test-context", "/api/v1/pods", []any{}, map[string]any{"labelSelector": nil}},
			want: FetchInput{Context: "test-context", Path: "/api/v1/pods", Fields: []string{}},
		},
		{
			name:      "selector not string",
			args:      []any{"test-context", "/api/v1/pods", []any{}, map[string]any{"labelSelector": 1.0}},
			wantError: "labelSelector must be a string",
		},
		{
			name:      "ctx not string",
			args:      []any{1.0, "/api/v1/pods", []any{}},
			wantError: "ctx must be a string",
		},
		{
			name:      "path not string",
			args:      []any{"test-context", nil, []any{}},
			wantError: "path must be a string",
		},
		{
			name:      "fields not array",
			args:      []any{"test-context", "/api/v1/pods", "metadata.name"},
			wantError: "fields",
		},
		{
			name:      "field not string",
			args:      []any{"test-context", "/api/v1/pods", []any{1.0}},
			wantError: "fields",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFetchInput(tt.args)
			if tt.wantError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
