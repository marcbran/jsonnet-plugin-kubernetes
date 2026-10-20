package kubernetes

import (
	"sort"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"k8s.io/client-go/tools/clientcmd"
)

type ContextOutput struct {
	Name      string `json:"name"`
	Current   bool   `json:"current"`
	Cluster   string `json:"cluster"`
	AuthInfo  string `json:"authInfo"`
	Namespace string `json:"namespace"`
}

func Contexts() jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "contexts",
		Params: ast.Identifiers{},
		Func: func(args []any) (any, error) {
			out, err := runContexts()
			if err != nil {
				return nil, err
			}
			return encodeOutput(out)
		},
	}
}

func runContexts() ([]ContextOutput, error) {
	cfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ContextOutput, 0, len(names))
	for _, name := range names {
		ctx := cfg.Contexts[name]
		out = append(out, ContextOutput{
			Name:      name,
			Current:   name == cfg.CurrentContext,
			Cluster:   ctx.Cluster,
			AuthInfo:  ctx.AuthInfo,
			Namespace: ctx.Namespace,
		})
	}
	return out, nil
}
