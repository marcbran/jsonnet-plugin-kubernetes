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
	authFor AuthByContext
}

func newClientRegistry(authFor AuthByContext) *clientRegistry {
	return &clientRegistry{clients: map[string]*clients{}, authFor: authFor}
}

func (c *clientRegistry) get(contextName string) (*clients, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cl, ok := c.clients[contextName]
	if !ok {
		var err error
		cl, err = BuildClients(contextName, c.authFor)
		if err != nil {
			return nil, fmt.Errorf("context %q: %w", contextName, err)
		}
		c.clients[contextName] = cl
	}
	return cl, nil
}

func BuildClients(contextName string, authFor AuthByContext) (*clients, error) {
	restConfig, err := BuildRestConfig(contextName, authFor)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &clients{dynamic: dyn}, nil
}

func BuildRestConfig(contextName string, authFor AuthByContext) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	restConfig.Timeout = 30 * time.Second
	if authFor == nil {
		return restConfig, nil
	}
	if contextName == "" {
		rawConfig, err := clientConfig.RawConfig()
		if err != nil {
			return nil, err
		}
		contextName = rawConfig.CurrentContext
	}
	auth, err := authFor(contextName)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	auth.apply(restConfig)
	return restConfig, nil
}
