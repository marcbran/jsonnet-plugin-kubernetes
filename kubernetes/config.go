package kubernetes

import (
	"fmt"
	"sync"
	"time"

	httpPlugin "github.com/marcbran/jsonnet-plugin-http/http"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type configCache struct {
	mu   sync.Mutex
	cfgs map[string]*httpPlugin.Config
}

func newConfigCache() *configCache {
	return &configCache{cfgs: map[string]*httpPlugin.Config{}}
}

func (c *configCache) get(contextName string) (*httpPlugin.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cfg, ok := c.cfgs[contextName]
	if !ok {
		var err error
		cfg, err = buildConfig(contextName)
		if err != nil {
			return nil, fmt.Errorf("context %q: %w", contextName, err)
		}
		c.cfgs[contextName] = cfg
	}
	return cfg, nil
}

func buildConfig(contextName string) (*httpPlugin.Config, error) {
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

	client, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return nil, err
	}

	return &httpPlugin.Config{
		BaseURL: restConfig.Host,
		Client:  client,
		Headers: map[string]string{},
	}, nil
}
