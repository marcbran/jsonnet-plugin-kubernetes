package kubernetes

import (
	"github.com/google/go-jsonnet"
	"github.com/marcbran/jpoet/pkg/jpoet"
)

func Plugin() *jpoet.Plugin {
	return jpoet.NewPlugin("kubernetes", []jsonnet.NativeFunction{
		Contexts(),
		Request(),
	})
}
