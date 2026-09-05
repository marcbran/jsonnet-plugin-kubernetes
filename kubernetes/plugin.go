package kubernetes

import (
	"github.com/google/go-jsonnet"
	"github.com/marcbran/jpoet/pkg/jpoet"
)

func Plugin() *jpoet.Plugin {
	clients := newClientRegistry()
	wc := newWatchCache(clients)
	return jpoet.NewPlugin(
		"kubernetes",
		[]jsonnet.NativeFunction{
			Contexts(),
			Get(wc),
			NeatGet(wc),
		},
		jpoet.WithWatchSource(wc),
	).WithCloser(wc)
}
