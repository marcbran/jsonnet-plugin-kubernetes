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
					k.get('test', '` + tc.path + `')
				`)

			then.
				the_error_contains(`secrets is forbidden`)
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
			k.get('test', '/api/v1/namespaces/default/secrets')
		`)

	then.
		the_error_contains(`secrets is forbidden`)
}
