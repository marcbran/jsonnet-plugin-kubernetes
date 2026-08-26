package kubernetesopenapi

import "context"

type FetchOpenAPIInput struct {
	Contexts []string `json:"contexts"`
	OutDir   string   `json:"outDir"`
}

type FetchOpenAPIOutput struct {
	OutDir        string   `json:"outDir"`
	BundleFile    string   `json:"bundleFile"`
	GroupVersions []string `json:"groupVersions"`
}

type Facade interface {
	FetchOpenAPI(ctx context.Context, in FetchOpenAPIInput) (FetchOpenAPIOutput, error)
}
