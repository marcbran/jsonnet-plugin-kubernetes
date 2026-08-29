package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func parsePath(path string) (gvr schema.GroupVersionResource, namespace string, name string, err error) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		return schema.GroupVersionResource{}, "", "", fmt.Errorf("invalid kubernetes path: %q", path)
	}
	switch segments[0] {
	case "api":
		return parseResourcePath(schema.GroupVersionResource{Version: segments[1]}, segments[2:], path)
	case "apis":
		if len(segments) < 4 {
			return schema.GroupVersionResource{}, "", "", fmt.Errorf("invalid kubernetes path: %q", path)
		}
		return parseResourcePath(schema.GroupVersionResource{Group: segments[1], Version: segments[2]}, segments[3:], path)
	default:
		return schema.GroupVersionResource{}, "", "", fmt.Errorf("invalid kubernetes path: %q", path)
	}
}

func parseResourcePath(gvr schema.GroupVersionResource, rest []string, path string) (schema.GroupVersionResource, string, string, error) {
	if len(rest) >= 3 && rest[0] == "namespaces" {
		namespace := rest[1]
		return finishResourcePath(gvr, namespace, rest[2:], path)
	}
	return finishResourcePath(gvr, "", rest, path)
}

func finishResourcePath(gvr schema.GroupVersionResource, namespace string, rest []string, path string) (schema.GroupVersionResource, string, string, error) {
	switch len(rest) {
	case 1:
		gvr.Resource = rest[0]
		return gvr, namespace, "", nil
	case 2:
		gvr.Resource = rest[0]
		return gvr, namespace, rest[1], nil
	default:
		return schema.GroupVersionResource{}, "", "", fmt.Errorf("invalid kubernetes path: %q", path)
	}
}
