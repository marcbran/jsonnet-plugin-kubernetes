package fetch

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	k8sclientset "k8s.io/client-go/kubernetes"
)

type Fetcher interface {
	Get(ctx context.Context, namespace, name string) (map[string]any, error)
	List(ctx context.Context, namespace string) (map[string]any, error)
}

type typedFactory func(typed k8sclientset.Interface) Fetcher

var typedFetchers = map[schema.GroupVersionResource]typedFactory{
	podsGVR: func(typed k8sclientset.Interface) Fetcher { return newPodsFetcher(typed) },
}

func For(dyn dynamic.Interface, typed k8sclientset.Interface, gvr schema.GroupVersionResource) Fetcher {
	if newFetcher, ok := typedFetchers[gvr]; ok {
		return newFetcher(typed)
	}
	return newDynamicFetcher(dyn, gvr)
}
