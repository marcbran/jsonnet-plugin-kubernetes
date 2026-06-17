package kubernetes

import (
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	httpPlugin "github.com/marcbran/jsonnet-plugin-http/http"
)

func Request() jsonnet.NativeFunction {
	cache := newConfigCache()

	return jsonnet.NativeFunction{
		Name:   "request",
		Params: ast.Identifiers{"input"},
		Func: func(args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("expected input object")
			}
			raw, ok := args[0].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("input must be an object")
			}
			contextName, _ := raw["context"].(string)

			cfg, err := cache.get(contextName)
			if err != nil {
				return nil, err
			}

			return httpPlugin.Request(cfg).Func(args)
		},
	}
}
