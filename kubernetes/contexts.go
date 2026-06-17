package kubernetes

import (
	"sort"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"k8s.io/client-go/tools/clientcmd"
)

func Contexts() jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "contexts",
		Params: ast.Identifiers{},
		Func: func(args []any) (any, error) {
			cfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
			if err != nil {
				return nil, err
			}
			names := make([]string, 0, len(cfg.Contexts))
			for name := range cfg.Contexts {
				names = append(names, name)
			}
			sort.Strings(names)
			rows := make([]any, 0, len(names))
			for _, name := range names {
				ctx := cfg.Contexts[name]
				rows = append(rows, map[string]any{
					"name":      name,
					"current":   name == cfg.CurrentContext,
					"cluster":   ctx.Cluster,
					"authInfo":  ctx.AuthInfo,
					"namespace": ctx.Namespace,
				})
			}
			return rows, nil
		},
	}
}
