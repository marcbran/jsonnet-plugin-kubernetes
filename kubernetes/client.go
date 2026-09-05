package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type clients struct {
	dynamic dynamic.Interface
}

type clientRegistry struct {
	mu      sync.Mutex
	clients map[string]*clients
}

func newClientRegistry() *clientRegistry {
	return &clientRegistry{clients: map[string]*clients{}}
}

func (c *clientRegistry) get(contextName string) (*clients, error) {
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

func BuildClients(contextName string) (*clients, error) {
	restConfig, err := BuildRestConfig(contextName)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &clients{dynamic: dyn}, nil
}

func BuildRestConfig(contextName string) (*rest.Config, error) {
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
