package kubernetesopenapi

import (
	"context"

	"github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/internal/infra/kubeconfig"
	"github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/internal/kubernetesopenapi/fetch"
	kubernetesopenapipkg "github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/pkg/kubernetesopenapi"
)

type facade struct{}

func NewFacade() kubernetesopenapipkg.Facade {
	return &facade{}
}

func (f *facade) FetchOpenAPI(ctx context.Context, in kubernetesopenapipkg.FetchOpenAPIInput) (kubernetesopenapipkg.FetchOpenAPIOutput, error) {
	getters := make([]fetch.Getter, 0, len(in.Contexts))
	for _, contextName := range in.Contexts {
		client, err := kubeconfig.NewClient(contextName)
		if err != nil {
			return kubernetesopenapipkg.FetchOpenAPIOutput{}, err
		}
		getters = append(getters, client)
	}
	out, err := fetch.Exec(ctx, getters, fetch.Input{
		OutDir: in.OutDir,
	})
	if err != nil {
		return kubernetesopenapipkg.FetchOpenAPIOutput{}, err
	}
	return kubernetesopenapipkg.FetchOpenAPIOutput{
		OutDir:        out.OutDir,
		BundleFile:    out.BundleFile,
		GroupVersions: out.GroupVersions,
	}, nil
}
