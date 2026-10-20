package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeOutput(t *testing.T) {
	tests := []struct {
		name   string
		output any
		want   any
	}{
		{
			name:   "exec",
			output: ExecOutput{ExitCode: 2, Stdout: "out", Stderr: "err"},
			want:   map[string]any{"exitCode": 2.0, "stdout": "out", "stderr": "err"},
		},
		{
			name: "proxy",
			output: ProxyOutput{
				Status:     404,
				StatusText: "Not Found",
				Headers:    map[string]string{"content-type": "application/json"},
				Body:       "{}",
			},
			want: map[string]any{
				"status":     404.0,
				"statusText": "Not Found",
				"headers":    map[string]any{"content-type": "application/json"},
				"body":       "{}",
			},
		},
		{
			name:   "contexts",
			output: []ContextOutput{{Name: "test-context", Current: true, Cluster: "test-context", AuthInfo: "admin"}},
			want: []any{map[string]any{
				"name":      "test-context",
				"current":   true,
				"cluster":   "test-context",
				"authInfo":  "admin",
				"namespace": "",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeOutput(tt.output)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
