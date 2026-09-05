package kubernetes

import (
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
)

func Get(watch *watchCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "get",
		Params: ast.Identifiers{"ctx", "path", "fields"},
		Func: func(args []any) (any, error) {
			return doFetch(watch, args, false)
		},
	}
}

func NeatGet(watch *watchCache) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "neatGet",
		Params: ast.Identifiers{"ctx", "path", "fields"},
		Func: func(args []any) (any, error) {
			return doFetch(watch, args, true)
		},
	}
}

func doFetch(watch *watchCache, args []any, neat bool) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("expected ctx, path, and fields")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("ctx must be a string")
	}
	path, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	fields, err := parseFields(args[2])
	if err != nil {
		return nil, err
	}

	gvr, namespace, name, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	if name != "" {
		out, err := watch.get(contextName, gvr, namespace, name)
		if err != nil {
			return nil, err
		}
		out = neatObject(out, neat)
		out = projectObject(out, fields)
		return out, nil
	}

	envelope, err := watch.list(contextName, gvr, namespace)
	if err != nil {
		return nil, err
	}
	envelope = neatList(envelope, neat)
	envelope = projectList(envelope, fields)
	return envelope, nil
}

func parseFields(arg any) ([]string, error) {
	raw, ok := arg.([]any)
	if !ok {
		return nil, fmt.Errorf("fields must be an array")
	}
	fields := make([]string, len(raw))
	for i, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("fields must be an array of strings")
		}
		fields[i] = s
	}
	return fields, nil
}
