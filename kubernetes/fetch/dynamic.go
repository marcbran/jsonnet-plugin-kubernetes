package fetch

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type dynamicFetcher struct {
	resource dynamic.NamespaceableResourceInterface
}

func newDynamicFetcher(dyn dynamic.Interface, gvr schema.GroupVersionResource) *dynamicFetcher {
	return &dynamicFetcher{resource: dyn.Resource(gvr)}
}

func (f *dynamicFetcher) Get(ctx context.Context, namespace, name string) (map[string]any, error) {
	var obj *unstructured.Unstructured
	var err error
	if namespace != "" {
		obj, err = f.resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = f.resource.Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, err
	}
	return obj.Object, nil
}

func (f *dynamicFetcher) List(ctx context.Context, namespace string) (map[string]any, error) {
	var list *unstructured.UnstructuredList
	var err error
	if namespace != "" {
		list, err = f.resource.Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = f.resource.List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}
	items := make([]any, len(list.Items))
	for i, item := range list.Items {
		items[i] = item.Object
	}
	envelope := make(map[string]any, len(list.Object)+1)
	for k, v := range list.Object {
		envelope[k] = v
	}
	envelope["items"] = items
	return envelope, nil
}
