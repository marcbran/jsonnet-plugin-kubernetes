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

			return httpPlugin.Request(cfg).Func([]any{map[string]any{
				"method": "GET",
				"path":   path,
			}})
		},
	}
}
