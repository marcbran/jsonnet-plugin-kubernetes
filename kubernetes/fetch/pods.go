package fetch

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclientset "k8s.io/client-go/kubernetes"
)

var podsGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

type podsFetcher struct {
	typed k8sclientset.Interface
}

func newPodsFetcher(typed k8sclientset.Interface) *podsFetcher {
	return &podsFetcher{typed: typed}
}

func (f *podsFetcher) Get(ctx context.Context, namespace, name string) (map[string]any, error) {
	pod, err := f.typed.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
}

func (f *podsFetcher) List(ctx context.Context, namespace string) (map[string]any, error) {
	list, err := f.typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]any, len(list.Items))
	for i := range list.Items {
		out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&list.Items[i])
		if err != nil {
			return nil, err
		}
		items[i] = out
	}
	metadata, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&list.ListMeta)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"apiVersion": "v1",
		"kind":       "PodList",
		"metadata":   metadata,
		"items":      items,
	}, nil
}
