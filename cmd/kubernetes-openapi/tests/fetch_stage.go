//go:build e2e

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kubernetespkg "github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/pkg/kubernetesopenapi"
	"github.com/stretchr/testify/require"
)

func (s *Stage) a_fake_kubernetes_cluster(contextName string, specsByKey map[string]map[string]any) *Stage {
	index := map[string]any{"paths": map[string]any{}}
	paths := index["paths"].(map[string]any)
	for key := range specsByKey {
		paths[key] = map[string]any{"serverRelativeURL": "/" + key + "/openapi.json"}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi/v3", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(index)
	})
	for key, spec := range specsByKey {
		path := "/" + key + "/openapi.json"
		spec := spec
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(spec)
		})
	}
	server := httptest.NewServer(mux)
	s.t.(*testing.T).Cleanup(server.Close)

	s.kubeconfigClusters = append(s.kubeconfigClusters, kubeconfigCluster{Context: contextName, Server: server.URL})
	s.contexts = append(s.contexts, contextName)
	s.writeKubeconfig()
	return s
}

func (s *Stage) writeKubeconfig() {
	var clusters, contexts strings.Builder
	for _, c := range s.kubeconfigClusters {
		clusterName := c.Context + "-cluster"
		fmt.Fprintf(&clusters, "- name: %s\n  cluster:\n    server: %s\n", clusterName, c.Server)
		fmt.Fprintf(&contexts, "- name: %s\n  context:\n    cluster: %s\n", c.Context, clusterName)
	}
	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
%susers: []
contexts:
%s`, clusters.String(), contexts.String())
	path := filepath.Join(s.tempDir, "kubeconfig")
	err := os.WriteFile(path, []byte(kubeconfig), 0644)
	require.NoError(s.t, err)
	s.t.(*testing.T).Setenv("KUBECONFIG", path)
}

func (s *Stage) a_fetch_output_under_temp(name string) *Stage {
	s.outDir = filepath.Join(s.tempDir, name)
	return s
}

func (s *Stage) the_fetch_command_is_run() *Stage {
	out, err := s.facade.FetchOpenAPI(context.Background(), kubernetespkg.FetchOpenAPIInput{
		Contexts: s.contexts,
		OutDir:   s.outDir,
	})
	if err != nil {
		s.lastOutput = out
		s.lastErr = err.Error()
		return s
	}
	s.lastOutput = out
	s.lastErr = ""
	return s
}

func (s *Stage) the_fetch_has_no_error() *Stage {
	require.Empty(s.t, s.lastErr)
	return s
}

func (s *Stage) the_fetch_group_versions_are(want []string) *Stage {
	require.ElementsMatch(s.t, want, s.lastOutput.GroupVersions)
	return s
}

func (s *Stage) the_bundle_file_contains(key string, expected map[string]any) *Stage {
	raw, err := os.ReadFile(s.lastOutput.BundleFile)
	require.NoError(s.t, err)
	var bundle map[string]any
	err = json.Unmarshal(raw, &bundle)
	require.NoError(s.t, err)
	require.Contains(s.t, bundle, key)
	require.Equal(s.t, expected, bundle[key])
	return s
}

func (s *Stage) a_per_group_spec_file_exists(dirName string) *Stage {
	require.FileExists(s.t, filepath.Join(s.outDir, dirName, "spec.json"))
	return s
}
