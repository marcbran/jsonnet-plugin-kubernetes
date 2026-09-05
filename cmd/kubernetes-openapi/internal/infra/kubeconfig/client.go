package kubeconfig

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/marcbran/jsonnet-plugin-kubernetes/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(contextName string) (*Client, error) {
	restConfig, err := kubernetes.BuildRestConfig(contextName)
	if err != nil {
		return nil, err
	}
	client, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return nil, err
	}
	return &Client{baseURL: restConfig.Host, http: client}, nil
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}
