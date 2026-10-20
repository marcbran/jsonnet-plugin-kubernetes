package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestParseProxyInput(t *testing.T) {
	tests := []struct {
		name      string
		args      []any
		want      ProxyInput
		wantError string
	}{
		{
			name: "service with defaults",
			args: []any{
				"test-context",
				map[string]any{"namespace": "search", "service": "opensearch"},
				"/_cluster/health",
				map[string]any{},
			},
			want: ProxyInput{
				Context: "test-context",
				Target:  ProxyTarget{Namespace: "search", Service: "opensearch"},
				Path:    "/_cluster/health",
				Options: ProxyOptions{Method: "GET"},
			},
		},
		{
			name: "pod with numeric port and https",
			args: []any{
				"test-context",
				map[string]any{"namespace": "search", "pod": "opensearch-0", "port": 9200.0, "scheme": "https"},
				"/_cat/indices?format=json",
				map[string]any{},
			},
			want: ProxyInput{
				Context: "test-context",
				Target:  ProxyTarget{Namespace: "search", Pod: "opensearch-0", Port: intstr.FromInt32(9200), Scheme: "https"},
				Path:    "/_cat/indices?format=json",
				Options: ProxyOptions{Method: "GET"},
			},
		},
		{
			name: "named port",
			args: []any{
				"test-context",
				map[string]any{"namespace": "nats", "service": "nats", "port": "monitor"},
				"/varz",
				map[string]any{},
			},
			want: ProxyInput{
				Context: "test-context",
				Target:  ProxyTarget{Namespace: "nats", Service: "nats", Port: intstr.FromString("monitor")},
				Path:    "/varz",
				Options: ProxyOptions{Method: "GET"},
			},
		},
		{
			name: "post with headers and body",
			args: []any{
				"test-context",
				map[string]any{"namespace": "search", "service": "opensearch", "port": 9200.0},
				"/_search",
				map[string]any{
					"method":  "post",
					"headers": map[string]any{"Content-Type": "application/json"},
					"body":    `{"query":{"match_all":{}}}`,
				},
			},
			want: ProxyInput{
				Context: "test-context",
				Target:  ProxyTarget{Namespace: "search", Service: "opensearch", Port: intstr.FromInt32(9200)},
				Path:    "/_search",
				Options: ProxyOptions{
					Method:  "POST",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    `{"query":{"match_all":{}}}`,
				},
			},
		},
		{
			name:      "wrong arg count",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch"}, "/"},
			wantError: "expected ctx, target, path, and options",
		},
		{
			name:      "ctx not string",
			args:      []any{1.0, map[string]any{"namespace": "search", "service": "opensearch"}, "/", map[string]any{}},
			wantError: "ctx must be a string",
		},
		{
			name:      "unknown target field",
			args:      []any{"test-context", map[string]any{"namespace": "search", "svc": "opensearch"}, "/", map[string]any{}},
			wantError: "unknown field",
		},
		{
			name:      "missing namespace",
			args:      []any{"test-context", map[string]any{"service": "opensearch"}, "/", map[string]any{}},
			wantError: "target.namespace is required",
		},
		{
			name:      "neither service nor pod",
			args:      []any{"test-context", map[string]any{"namespace": "search"}, "/", map[string]any{}},
			wantError: "exactly one of service or pod",
		},
		{
			name:      "both service and pod",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch", "pod": "opensearch-0"}, "/", map[string]any{}},
			wantError: "exactly one of service or pod",
		},
		{
			name:      "bad scheme",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch", "scheme": "ftp"}, "/", map[string]any{}},
			wantError: "target.scheme must be http or https",
		},
		{
			name:      "path not string",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch"}, nil, map[string]any{}},
			wantError: "path must be a string",
		},
		{
			name:      "invalid path",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch"}, "/%zz", map[string]any{}},
			wantError: "path",
		},
		{
			name:      "unknown option",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch"}, "/", map[string]any{"query": map[string]any{}}},
			wantError: "unknown field",
		},
		{
			name:      "header not string",
			args:      []any{"test-context", map[string]any{"namespace": "search", "service": "opensearch"}, "/", map[string]any{"headers": map[string]any{"X-Count": 1.0}}},
			wantError: "options",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProxyInput(tt.args)
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

func TestProxyTargetSegment(t *testing.T) {
	tests := []struct {
		name   string
		target ProxyTarget
		want   string
	}{
		{name: "name only", target: ProxyTarget{Service: "opensearch"}, want: "opensearch"},
		{name: "numeric port", target: ProxyTarget{Service: "opensearch", Port: intstr.FromInt32(9200)}, want: "opensearch:9200"},
		{name: "named port", target: ProxyTarget{Pod: "nats-0", Port: intstr.FromString("monitor")}, want: "nats-0:monitor"},
		{name: "https with port", target: ProxyTarget{Service: "opensearch", Port: intstr.FromInt32(9200), Scheme: "https"}, want: "https:opensearch:9200"},
		{name: "https without port", target: ProxyTarget{Service: "opensearch", Scheme: "https"}, want: "https:opensearch:"},
		{name: "explicit http", target: ProxyTarget{Service: "opensearch", Port: intstr.FromInt32(80), Scheme: "http"}, want: "opensearch:80"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.target.segment())
		})
	}
}
