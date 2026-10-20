//go:build e2e

package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stretchr/testify/require"
)

type fakeCluster struct {
	context string
	server  *httptest.Server

	mu          sync.Mutex
	collections map[string][]map[string]any
	forbidden   map[string]bool
}

func (s *Stage) a_cluster(contextName string) *Stage {
	c := &fakeCluster{
		context:     contextName,
		collections: map[string][]map[string]any{},
		forbidden:   map[string]bool{},
	}
	c.server = httptest.NewServer(http.HandlerFunc(c.serve))
	s.t.Cleanup(func() {
		c.server.CloseClientConnections()
		c.server.Close()
	})
	s.clusters = append(s.clusters, c)
	s.writeKubeconfig()
	return s
}

func (s *Stage) the_cluster_has(collectionPath string, objects ...string) *Stage {
	c := s.currentCluster()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, raw := range objects {
		var obj map[string]any
		err := json.Unmarshal([]byte(raw), &obj)
		require.NoError(s.t, err)
		c.collections[collectionPath] = append(c.collections[collectionPath], obj)
	}
	return s
}

func (s *Stage) the_cluster_forbids(collectionPath string) *Stage {
	c := s.currentCluster()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forbidden[collectionPath] = true
	return s
}

func (s *Stage) currentCluster() *fakeCluster {
	require.NotEmpty(s.t, s.clusters, "a_cluster must be called first")
	return s.clusters[len(s.clusters)-1]
}

func (s *Stage) writeKubeconfig() {
	var clusters, users, contexts strings.Builder
	for _, c := range s.clusters {
		fmt.Fprintf(&clusters, "- name: %s\n  cluster:\n    server: %s\n", c.context, c.server.URL)
		fmt.Fprintf(&users, "- name: %s\n  user:\n    token: e2e\n", c.context)
		fmt.Fprintf(&contexts, "- name: %s\n  context:\n    cluster: %s\n    user: %s\n", c.context, c.context, c.context)
	}
	kubeconfig := fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n%susers:\n%scontexts:\n%s",
		clusters.String(), users.String(), contexts.String())
	kubeconfigPath := filepath.Join(s.tempDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0o600)
	require.NoError(s.t, err)
	s.t.Setenv("KUBECONFIG", kubeconfigPath)
}

func (c *fakeCluster) serve(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	forbidden := c.forbidden[r.URL.Path]
	objects, found := c.collections[r.URL.Path]
	c.mu.Unlock()

	if forbidden {
		writeStatus(w, http.StatusForbidden, "Forbidden",
			fmt.Sprintf("%s is forbidden: User \"e2e\" cannot %s resource %q", path.Base(r.URL.Path), verb(r), path.Base(r.URL.Path)))
		return
	}
	if !found {
		writeStatus(w, http.StatusNotFound, "NotFound", fmt.Sprintf("the server could not find the requested resource (%s)", r.URL.Path))
		return
	}
	objects = filterByFieldSelector(objects, r.URL.Query().Get("fieldSelector"))
	if verb(r) == "watch" {
		c.serveWatch(w, r, objects)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"apiVersion": "v1",
		"kind":       "List",
		"metadata":   map[string]any{"resourceVersion": "1"},
		"items":      objects,
	})
}

func (c *fakeCluster) serveWatch(w http.ResponseWriter, r *http.Request, objects []map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	if r.URL.Query().Get("sendInitialEvents") == "true" {
		for _, obj := range objects {
			_ = enc.Encode(map[string]any{"type": "ADDED", "object": obj})
		}
		_ = enc.Encode(map[string]any{"type": "BOOKMARK", "object": initialEventsEndBookmark(r.URL.Path, objects)})
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	<-r.Context().Done()
}

func initialEventsEndBookmark(collectionPath string, objects []map[string]any) map[string]any {
	apiVersion, kind := "v1", "Unknown"
	if len(objects) > 0 {
		apiVersion, _ = objects[0]["apiVersion"].(string)
		kind, _ = objects[0]["kind"].(string)
	}
	return map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"resourceVersion": "1",
			"annotations":     map[string]any{"k8s.io/initial-events-end": "true"},
		},
	}
}

func filterByFieldSelector(objects []map[string]any, selector string) []map[string]any {
	name, ok := strings.CutPrefix(selector, "metadata.name=")
	if !ok {
		return objects
	}
	var filtered []map[string]any
	for _, obj := range objects {
		metadata, _ := obj["metadata"].(map[string]any)
		if metadata["name"] == name {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}

func verb(r *http.Request) string {
	watch := r.URL.Query().Get("watch")
	if watch == "true" || watch == "1" {
		return "watch"
	}
	return "list"
}

func writeStatus(w http.ResponseWriter, code int, reason, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"apiVersion": "v1",
		"kind":       "Status",
		"status":     "Failure",
		"message":    message,
		"reason":     reason,
		"code":       code,
	})
}
