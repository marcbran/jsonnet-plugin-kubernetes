package kubernetes

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type clientCache struct {
	mu      sync.Mutex
	clients map[string]dynamic.Interface
}

func newClientCache() *clientCache {
	return &clientCache{clients: map[string]dynamic.Interface{}}
}

func (c *clientCache) get(contextName string) (dynamic.Interface, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	client, ok := c.clients[contextName]
	if !ok {
		var err error
		client, err = BuildClient(contextName)
		if err != nil {
			return nil, fmt.Errorf("context %q: %w", contextName, err)
		}
		c.clients[contextName] = client
	}
	return client, nil
}

func buildRestConfig(contextName string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}
	restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	restConfig.Timeout = 30 * time.Second
	return restConfig, nil
}

func BuildClient(contextName string) (dynamic.Interface, error) {
	restConfig, err := buildRestConfig(contextName)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(restConfig)
}

func BuildHTTPClient(contextName string) (*http.Client, string, error) {
	restConfig, err := buildRestConfig(contextName)
	if err != nil {
		return nil, "", err
	}
	client, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return nil, "", err
	}
	return client, restConfig.Host, nil
}
