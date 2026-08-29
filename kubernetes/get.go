package kubernetes

import (
	"context"
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Get(cache *clientCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "get",
		Params: ast.Identifiers{"ctx", "path"},
		Func: func(args []any) (any, error) {
			return fetch(cache, args, false)
		},
	}
}

func NeatGet(cache *clientCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "neatGet",
		Params: ast.Identifiers{"ctx", "path"},
		Func: func(args []any) (any, error) {
			return fetch(cache, args, true)
		},
	}
}

func fetch(cache *clientCache, args []any, neat bool) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("expected ctx and path")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("ctx must be a string")
	}
	path, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	client, err := cache.get(contextName)
	if err != nil {
		return nil, err
	}
	gvr, namespace, name, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	resource := client.Resource(gvr)
	ctx := context.Background()

	if name != "" {
		var obj *unstructured.Unstructured
		if namespace != "" {
			obj, err = resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		} else {
			obj, err = resource.Get(ctx, name, metav1.GetOptions{})
		}
		if err != nil {
			return nil, err
		}
		out := obj.Object
		if neat {
			out = stripManagedFields(out)
		}
		return out, nil
	}

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = resource.Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = resource.List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}
	items := make([]any, len(list.Items))
	for i, item := range list.Items {
		out := item.Object
		if neat {
			out = stripManagedFields(out)
		}
		items[i] = out
	}
	envelope := make(map[string]any, len(list.Object)+1)
	for k, v := range list.Object {
		envelope[k] = v
	}
	envelope["items"] = items
	return envelope, nil
}
