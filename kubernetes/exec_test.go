package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExecInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []any
		want      ExecInput
		wantError string
	}{
		{
			name: "without container",
			args: []any{
				"test-context",
				map[string]any{"namespace": "search", "pod": "opensearch-0"},
				[]any{"echo", "hi"},
			},
			want: ExecInput{
				Context: "test-context",
				Target:  ExecTarget{Namespace: "search", Pod: "opensearch-0"},
				Command: []string{"echo", "hi"},
			},
		},
		{
			name: "with container",
			args: []any{
				"test-context",
				map[string]any{"namespace": "search", "pod": "opensearch-0", "container": "opensearch"},
				[]any{"sh", "-c", "curl -s localhost:9200"},
			},
			want: ExecInput{
				Context: "test-context",
				Target:  ExecTarget{Namespace: "search", Pod: "opensearch-0", Container: "opensearch"},
				Command: []string{"sh", "-c", "curl -s localhost:9200"},
			},
		},
		{
			name:      "wrong arg count",
			args:      []any{"test-context", map[string]any{"namespace": "search", "pod": "opensearch-0"}},
			wantError: "expected ctx, target, and command",
		},
		{
			name:      "ctx not string",
			args:      []any{nil, map[string]any{"namespace": "search", "pod": "opensearch-0"}, []any{"echo"}},
			wantError: "ctx must be a string",
		},
		{
			name:      "target not object",
			args:      []any{"test-context", "opensearch-0", []any{"echo"}},
			wantError: "target",
		},
		{
			name:      "unknown target field",
			args:      []any{"test-context", map[string]any{"namespace": "search", "pod": "opensearch-0", "contaner": "x"}, []any{"echo"}},
			wantError: "unknown field",
		},
		{
			name:      "missing namespace",
			args:      []any{"test-context", map[string]any{"pod": "opensearch-0"}, []any{"echo"}},
			wantError: "target.namespace is required",
		},
		{
			name:      "missing pod",
			args:      []any{"test-context", map[string]any{"namespace": "search"}, []any{"echo"}},
			wantError: "target.pod is required",
		},
		{
			name:      "command not array",
			args:      []any{"test-context", map[string]any{"namespace": "search", "pod": "opensearch-0"}, "echo hi"},
			wantError: "command",
		},
		{
			name:      "command element not string",
			args:      []any{"test-context", map[string]any{"namespace": "search", "pod": "opensearch-0"}, []any{"echo", 1.0}},
			wantError: "command",
		},
		{
			name:      "empty command",
			args:      []any{"test-context", map[string]any{"namespace": "search", "pod": "opensearch-0"}, []any{}},
			wantError: "command must not be empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExecInput(tt.args)
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
