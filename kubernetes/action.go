package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/marcbran/jpoet/pkg/jpoet"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

const restartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

var patchTypes = map[string]types.PatchType{
	"strategic": types.StrategicMergePatchType,
	"merge":     types.MergePatchType,
	"json":      types.JSONPatchType,
}

func Patch(clients *clientRegistry) jpoet.ActionFunc {
	return func(ctx context.Context, data map[string]any) (string, error) {
		patchType, err := resolvePatchType(data)
		if err != nil {
			return "", err
		}
		patch, ok := data["patch"]
		if !ok {
			return "", fmt.Errorf("patch is required")
		}
		body, err := json.Marshal(patch)
		if err != nil {
			return "", err
		}
		resource, name, err := resolveResource(clients, data)
		if err != nil {
			return "", err
		}
		return applyPatch(ctx, resource, name, patchType, body)
	}
}

func Restart(clients *clientRegistry) jpoet.ActionFunc {
	return func(ctx context.Context, data map[string]any) (string, error) {
		resource, name, err := resolveResource(clients, data)
		if err != nil {
			return "", err
		}
		body, err := json.Marshal(map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"metadata": map[string]any{
						"annotations": map[string]any{
							restartedAtAnnotation: time.Now().Format(time.RFC3339),
						},
					},
				},
			},
		})
		if err != nil {
			return "", err
		}
		return applyPatch(ctx, resource, name, types.StrategicMergePatchType, body)
	}
}

func Delete(clients *clientRegistry) jpoet.ActionFunc {
	return func(ctx context.Context, data map[string]any) (string, error) {
		resource, name, err := resolveResource(clients, data)
		if err != nil {
			return "", err
		}
		err = resource.Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("deleted %s", name), nil
	}
}

func applyPatch(ctx context.Context, resource dynamic.ResourceInterface, name string, patchType types.PatchType, body []byte) (string, error) {
	out, err := resource.Patch(ctx, name, patchType, body, metav1.PatchOptions{})
	if err != nil {
		return "", err
	}
	result, err := json.Marshal(out.Object)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func resolveResource(clients *clientRegistry, data map[string]any) (dynamic.ResourceInterface, string, error) {
	contextName, err := stringField(data, "ctx")
	if err != nil {
		return nil, "", err
	}
	path, err := stringField(data, "path")
	if err != nil {
		return nil, "", err
	}
	gvr, namespace, name, err := parsePath(path)
	if err != nil {
		return nil, "", err
	}
	if name == "" {
		return nil, "", fmt.Errorf("path must name a resource: %q", path)
	}
	cl, err := clients.get(contextName)
	if err != nil {
		return nil, "", err
	}
	if namespace == "" {
		return cl.dynamic.Resource(gvr), name, nil
	}
	return cl.dynamic.Resource(gvr).Namespace(namespace), name, nil
}

func resolvePatchType(data map[string]any) (types.PatchType, error) {
	raw, ok := data["type"]
	if !ok {
		return types.StrategicMergePatchType, nil
	}
	name, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("type must be a string")
	}
	patchType, ok := patchTypes[name]
	if !ok {
		return "", fmt.Errorf("unknown patch type: %s", name)
	}
	return patchType, nil
}

func stringField(data map[string]any, key string) (string, error) {
	raw, ok := data[key]
	if !ok {
		return "", fmt.Errorf("%s is required", key)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return value, nil
}
