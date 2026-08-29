package kubernetes

import (
	"context"
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"

	"github.com/marcbran/jsonnet-plugin-kubernetes/kubernetes/fetch"
)

func Get(cache *clientCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "get",
		Params: ast.Identifiers{"ctx", "path"},
		Func: func(args []any) (any, error) {
			return doFetch(cache, args, false)
		},
	}
}

func NeatGet(cache *clientCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "neatGet",
		Params: ast.Identifiers{"ctx", "path"},
		Func: func(args []any) (any, error) {
			return doFetch(cache, args, true)
		},
	}
}

func doFetch(cache *clientCache, args []any, neat bool) (any, error) {
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

	cl, err := cache.get(contextName)
	if err != nil {
		return nil, err
	}
	gvr, namespace, name, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	fetcher := fetch.For(cl.dynamic, cl.typed, gvr)
	ctx := context.Background()

	if name != "" {
		out, err := fetcher.Get(ctx, namespace, name)
		if err != nil {
			return nil, err
		}
		return neatObject(out, neat), nil
	}

	envelope, err := fetcher.List(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return neatList(envelope, neat), nil
}
