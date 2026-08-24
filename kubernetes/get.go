package kubernetes

import (
	"fmt"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	httpPlugin "github.com/marcbran/jsonnet-plugin-http/http"
)

func Get() jsonnet.NativeFunction {
	cache := newConfigCache()

	return jsonnet.NativeFunction{
		Name:   "get",
		Params: ast.Identifiers{"ctx", "path"},
		Func: func(args []any) (any, error) {
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

			cfg, err := cache.get(contextName)
			if err != nil {
				return nil, err
			}

			out, err := httpPlugin.Request(cfg).Func([]any{map[string]any{
				"method": "GET",
				"path":   path,
			}})
			if err != nil {
				return nil, err
			}
			envelope, ok := out.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("unexpected response shape")
			}
			return envelope["body"], nil
		},
	}
}
