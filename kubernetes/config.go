package kubernetes

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	k8sclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type clients struct {
	dynamic dynamic.Interface
	typed   k8sclientset.Interface
}

type clientCache struct {
	mu      sync.Mutex
	clients map[string]*clients
}

func newClientCache() *clientCache {
	return &clientCache{clients: map[string]*clients{}}
}

func (c *clientCache) get(contextName string) (*clients, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cl, ok := c.clients[contextName]
	if !ok {
		var err error
		cl, err = BuildClients(contextName)
		if err != nil {
			return nil, fmt.Errorf("context %q: %w", contextName, err)
		}
		c.clients[contextName] = cl
	}
	return cl, nil
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

func BuildClients(contextName string) (*clients, error) {
	restConfig, err := buildRestConfig(contextName)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	typed, err := k8sclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &clients{dynamic: dyn, typed: typed}, nil
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
