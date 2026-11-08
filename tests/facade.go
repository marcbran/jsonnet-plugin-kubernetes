//go:build e2e

package tests

import (
	"bytes"
	"context"
	"errors"

	"github.com/marcbran/jpoet/pkg/jpoet"
	"github.com/marcbran/jpoet/pkg/watch"
	"github.com/marcbran/jsonnet-plugin-kubernetes/kubernetes"
)

type Result = watch.Result

type Facade interface {
	Evaluate(ctx context.Context, snippet string) (string, error)
	Watch(ctx context.Context, snippet string) (Result, <-chan Result, func(), error)
	Close() error
}

type JpoetFacade struct {
	plugin *jpoet.Plugin
	env    *watch.Environment
}

func NewE2EFacade() (Facade, error) {
	plugin := kubernetes.Plugin()
	env, err := watch.New(jpoet.Env(
		jpoet.EnvWithPlugin(plugin),
		jpoet.EnvFileImport([]string{"../pkg/jsonnet-plugin-kubernetes"}),
	))
	if err != nil {
		return nil, errors.Join(err, plugin.Close())
	}
	return &JpoetFacade{plugin: plugin, env: env}, nil
}

func (f *JpoetFacade) Evaluate(ctx context.Context, snippet string) (string, error) {
	var out bytes.Buffer
	err := f.env.Eval(ctx,
		jpoet.EvalSnippetInput("main.jsonnet", snippet),
		jpoet.EvalWriterOutput(&out),
	)
	return out.String(), err
}

func (f *JpoetFacade) Watch(ctx context.Context, snippet string) (Result, <-chan Result, func(), error) {
	var initial Result
	var updates <-chan Result
	unsubscribe, err := f.env.Watch(ctx,
		watch.WatchWithKey("main"),
		watch.WatchSnippetInput("main.jsonnet", snippet),
		watch.WatchValueOutput(&initial, &updates),
	)
	if unsubscribe == nil {
		unsubscribe = func() {}
	}
	return initial, updates, unsubscribe, err
}

func (f *JpoetFacade) Close() error {
	return errors.Join(f.plugin.Close(), f.env.Close())
}
