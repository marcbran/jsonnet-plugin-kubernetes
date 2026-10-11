package kubernetes

import (
	"github.com/google/go-jsonnet"
	"github.com/marcbran/jpoet/pkg/jpoet"
)

type Config struct {
	AuthByContext AuthByContext
	pluginOpts    []jpoet.PluginOption
}

type Option func(*Config)

func WithAuthByContext(authFor AuthByContext) Option {
	return func(c *Config) {
		c.AuthByContext = authFor
	}
}

func WithPluginOption(opt jpoet.PluginOption) Option {
	return func(c *Config) {
		c.pluginOpts = append(c.pluginOpts, opt)
	}
}

func WithMiddleware(middleware ...jpoet.Middleware) Option {
	return WithPluginOption(jpoet.WithMiddleware(middleware...))
}

func WithHook(hook jpoet.InvokeHook) Option {
	return WithPluginOption(jpoet.WithHook(hook))
}

func Plugin(opts ...Option) *jpoet.Plugin {
	cfg := &Config{}
	for _, o := range opts {
		o(cfg)
	}
	clients := newClientRegistry(cfg.AuthByContext)
	wc := newWatchCache(clients)
	allOpts := append([]jpoet.PluginOption{
		jpoet.WithWatchSource(wc),
		jpoet.WithCloser(wc),
		jpoet.WithAction("patch", Patch(clients)),
		jpoet.WithAction("delete", Delete(clients)),
		jpoet.WithAction("restart", Restart(clients)),
	}, cfg.pluginOpts...)
	return jpoet.NewPlugin(
		"kubernetes",
		[]jsonnet.NativeFunction{
			Contexts(),
			Get(wc),
			NeatGet(wc),
		},
		allOpts...,
	)
}
