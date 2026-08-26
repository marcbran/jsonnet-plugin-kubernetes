//go:build e2e

package tests

import "testing"

func TestFetch(t *testing.T) {
	given, when, then := scenario(t)

	coreSpec := map[string]any{
		"paths": map[string]any{
			"/api/v1/namespaces/{namespace}/widgets": map[string]any{
				"get": map[string]any{"x-kubernetes-group-version-kind": map[string]any{"kind": "Widget"}},
			},
		},
	}
	appsSpec := map[string]any{
		"paths": map[string]any{
			"/apis/apps/v1/deployments": map[string]any{
				"get": map[string]any{"x-kubernetes-group-version-kind": map[string]any{"kind": "Deployment"}},
			},
		},
	}

	given.
		a_fake_kubernetes_cluster("test", map[string]map[string]any{
			"api/v1":       coreSpec,
			"apis/apps/v1": appsSpec,
		}).and().
		a_fetch_output_under_temp("kubernetes")

	when.
		the_fetch_command_is_run()

	then.
		the_fetch_has_no_error().and().
		the_fetch_group_versions_are([]string{"api/v1", "apis/apps/v1"}).and().
		the_bundle_file_contains("api/v1", coreSpec).and().
		the_bundle_file_contains("apis/apps/v1", appsSpec).and().
		a_per_group_spec_file_exists("api__v1").and().
		a_per_group_spec_file_exists("apis__apps__v1")
}

func TestFetchMultipleContexts(t *testing.T) {
	given, when, then := scenario(t)

	coreSpec := map[string]any{
		"paths": map[string]any{
			"/api/v1/namespaces/{namespace}/widgets": map[string]any{
				"get": map[string]any{"x-kubernetes-group-version-kind": map[string]any{"kind": "Widget"}},
			},
		},
	}
	crdSpec := map[string]any{
		"paths": map[string]any{
			"/apis/acme.io/v1/gadgets": map[string]any{
				"get": map[string]any{"x-kubernetes-group-version-kind": map[string]any{"kind": "Gadget"}},
			},
		},
	}

	given.
		a_fake_kubernetes_cluster("cluster-a", map[string]map[string]any{
			"api/v1": coreSpec,
		}).and().
		a_fake_kubernetes_cluster("cluster-b", map[string]map[string]any{
			"apis/acme.io/v1": crdSpec,
		}).and().
		a_fetch_output_under_temp("kubernetes")

	when.
		the_fetch_command_is_run()

	then.
		the_fetch_has_no_error().and().
		the_fetch_group_versions_are([]string{"api/v1", "apis/acme.io/v1"}).and().
		the_bundle_file_contains("api/v1", coreSpec).and().
		the_bundle_file_contains("apis/acme.io/v1", crdSpec)
}
