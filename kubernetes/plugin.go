package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	nativehttp "net/http"
	"sync"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/marcbran/jpoet/pkg/jpoet"
	httpPlugin "github.com/marcbran/jsonnet-plugin-http/http"
	"k8s.io/client-go/tools/clientcmd"
)

func Plugin() *jpoet.Plugin {
	var mu sync.Mutex
	cfgs := map[string]*httpPlugin.Config{}

	return jpoet.NewPlugin("kubernetes", []jsonnet.NativeFunction{
		{
			Name:   "request",
			Params: ast.Identifiers{"input"},
			Func: func(args []any) (any, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("expected input object")
				}
				raw, ok := args[0].(map[string]any)
				if !ok {
					return nil, fmt.Errorf("input must be an object")
				}
				contextName, _ := raw["context"].(string)

				mu.Lock()
				cfg, ok := cfgs[contextName]
				if !ok {
					var err error
					cfg, err = buildConfig(contextName)
					if err != nil {
						mu.Unlock()
						return nil, fmt.Errorf("context %q: %w", contextName, err)
					}
					cfgs[contextName] = cfg
				}
				mu.Unlock()

				return httpPlugin.Request(cfg).Func(args)
			},
		},
	})
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

	tlsCfg := &tls.Config{}
	if len(restConfig.TLSClientConfig.CAData) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(restConfig.TLSClientConfig.CAData)
		tlsCfg.RootCAs = pool
	}
	if len(restConfig.TLSClientConfig.CertData) > 0 && len(restConfig.TLSClientConfig.KeyData) > 0 {
		cert, err := tls.X509KeyPair(restConfig.TLSClientConfig.CertData, restConfig.TLSClientConfig.KeyData)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	cfg := &httpPlugin.Config{
		BaseURL: restConfig.Host,
		Client: &nativehttp.Client{
			Timeout: 30 * time.Second,
			Transport: &nativehttp.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
		Headers: map[string]string{},
	}
	if restConfig.BearerToken != "" {
		cfg.Headers["Authorization"] = "Bearer " + restConfig.BearerToken
	}
	return cfg, nil
}
