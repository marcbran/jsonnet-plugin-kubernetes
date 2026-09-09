package kubernetes

import (
	"github.com/google/go-jsonnet"
	"github.com/marcbran/jpoet/pkg/jpoet"
)

func Plugin(opts ...jpoet.PluginOption) *jpoet.Plugin {
	clients := newClientRegistry()
	wc := newWatchCache(clients)
	allOpts := append([]jpoet.PluginOption{
		jpoet.WithWatchSource(wc),
		jpoet.WithCloser(wc),
	}, opts...)
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
