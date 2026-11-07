package kubernetes

import (
	"context"
	"fmt"

	"github.com/marcbran/jpoet/pkg/jpoet"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type FetchInput struct {
	Context string
	Path    string
	Fields  []string
}

func Get(watch *watchCache) jpoet.Function {
	return jpoet.Function{
		Name: "get",
		Func: func(ctx context.Context, args []any) (any, error) {
			fi, err := parseFetchInput(args)
			if err != nil {
				return nil, invalidArgs(err)
			}
			out, err := runFetch(ctx, watch, fi, false)
			return out, failure(err)
		},
	}
}

func NeatGet(watch *watchCache) jpoet.Function {
	return jpoet.Function{
		Name: "neatGet",
		Func: func(ctx context.Context, args []any) (any, error) {
			fi, err := parseFetchInput(args)
			if err != nil {
				return nil, invalidArgs(err)
			}
			out, err := runFetch(ctx, watch, fi, true)
			return out, failure(err)
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

func runFetch(ctx context.Context, watch *watchCache, fi FetchInput, neat bool) (any, error) {
	gvr, namespace, name, err := parsePath(fi.Path)
	if err != nil {
		return nil, apierrors.NewBadRequest(err.Error())
	}

	if name != "" {
		out, err := watch.get(ctx, fi.Context, gvr, namespace, name)
		if err != nil {
			return nil, err
		}
		out = neatObject(out, neat)
		out = projectObject(out, fi.Fields)
		return out, nil
	}

	envelope, err := watch.list(ctx, fi.Context, gvr, namespace)
	if err != nil {
		return nil, err
	}
	envelope = neatList(envelope, neat)
	envelope = projectList(envelope, fi.Fields)
	return envelope, nil
}
