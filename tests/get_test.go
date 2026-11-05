//go:build e2e

package tests

import "testing"

const configMapA = `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"a","namespace":"default"},"data":{"key":"value-a"}}`
const configMapB = `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"b","namespace":"default"},"data":{"key":"value-b"}}`

func TestGetList(t *testing.T) {
	given, when, then := scenario(t)

	given.
		a_cluster("test").and().
		the_cluster_has("/api/v1/namespaces/default/configmaps", configMapA, configMapB)

	when.
		the_jsonnet_is_evaluated(`
			local k = import 'main.libsonnet';
			[item.data.key for item in k.get('test', '/api/v1/namespaces/default/configmaps').items]
		`)

	then.
		there_is_no_error().and().
		the_output_is(`["value-a", "value-b"]`)
}

func TestGetSingleResource(t *testing.T) {
	given, when, then := scenario(t)

	given.
		a_cluster("test").and().
		the_cluster_has("/api/v1/namespaces/default/configmaps", configMapA, configMapB)

	when.
		the_jsonnet_is_evaluated(`
			local k = import 'main.libsonnet';
			k.get('test', '/api/v1/namespaces/default/configmaps/b').data
		`)

	then.
		there_is_no_error().and().
		the_output_is(`{"key": "value-b"}`)
}

func TestGetSingleResourceNotFound(t *testing.T) {
	given, when, then := scenario(t)

	given.
		a_cluster("test").and().
		the_cluster_has("/api/v1/namespaces/default/configmaps", configMapA)

	when.
		the_jsonnet_is_evaluated(`
			local k = import 'main.libsonnet';
			local r = k.get('test', '/api/v1/namespaces/default/configmaps/missing');
			r._failure
		`)

	then.
		there_is_no_error().and().
		the_output_is(`{"code": 404, "reason": "NotFound", "message": "configmaps \"missing\" not found"}`)
}

func TestGetForbidden(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{name: "list", path: "/api/v1/namespaces/default/secrets"},
		{name: "single resource", path: "/api/v1/namespaces/default/secrets/a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			given, when, then := scenario(t)

			given.
				a_cluster("test").and().
				the_cluster_forbids("/api/v1/namespaces/default/secrets")

			when.
				the_jsonnet_is_evaluated(`
					local k = import 'main.libsonnet';
					local r = k.get('test', '` + tc.path + `');
					{ code: r._failure.code, reason: r._failure.reason, forbidden: std.length(std.findSubstr('secrets is forbidden', r._failure.message)) > 0 }
				`)

			then.
				there_is_no_error().and().
				the_output_is(`{"code": 403, "reason": "Forbidden", "forbidden": true}`)
		})
	}
}

func TestWatchGetForbidden(t *testing.T) {
	given, when, then := scenario(t)

	given.
		a_cluster("test").and().
		the_cluster_forbids("/api/v1/namespaces/default/secrets")

	when.
		the_jsonnet_is_watched(`
			local k = import 'main.libsonnet';
			local r = k.get('test', '/api/v1/namespaces/default/secrets');
			{ code: r._failure.code, reason: r._failure.reason, forbidden: std.length(std.findSubstr('secrets is forbidden', r._failure.message)) > 0 }
		`)

	then.
		there_is_no_error().and().
		the_output_is(`{"code": 403, "reason": "Forbidden", "forbidden": true}`)
}
