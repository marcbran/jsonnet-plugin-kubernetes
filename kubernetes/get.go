package kubernetes

import (
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

type FetchInput struct {
	Context string
	Path    string
	Fields  []string
}

func Get(watch *watchCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "get",
		Params: ast.Identifiers{"ctx", "path", "fields"},
		Func: func(args []any) (any, error) {
			fi, err := parseFetchInput(args)
			if err != nil {
				return nil, err
			}
			return runFetch(watch, fi, false)
		},
	}
}

func NeatGet(watch *watchCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "neatGet",
		Params: ast.Identifiers{"ctx", "path", "fields"},
		Func: func(args []any) (any, error) {
			fi, err := parseFetchInput(args)
			if err != nil {
				return nil, err
			}
			return runFetch(watch, fi, true)
		},
	}
}

func parseFetchInput(args []any) (FetchInput, error) {
	if len(args) != 3 {
		return FetchInput{}, fmt.Errorf("expected ctx, path, and fields")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return FetchInput{}, fmt.Errorf("ctx must be a string")
	}
	path, ok := args[1].(string)
	if !ok {
		return FetchInput{}, fmt.Errorf("path must be a string")
	}
	fields, err := decodeArg[[]string](args[2], "fields")
	if err != nil {
		return FetchInput{}, err
	}
	return FetchInput{Context: contextName, Path: path, Fields: fields}, nil
}

func runFetch(watch *watchCache, fi FetchInput, neat bool) (any, error) {
	gvr, namespace, name, err := parsePath(fi.Path)
	if err != nil {
		return nil, err
	}

	if name != "" {
		out, err := watch.get(fi.Context, gvr, namespace, name)
		if err != nil {
			return nil, err
		}
		out = neatObject(out, neat)
		out = projectObject(out, fi.Fields)
		return out, nil
	}

	envelope, err := watch.list(fi.Context, gvr, namespace)
	if err != nil {
		return nil, err
	}
	envelope = neatList(envelope, neat)
	envelope = projectList(envelope, fi.Fields)
	return envelope, nil
}
