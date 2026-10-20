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
		jpoet.EnvFileImport([]string{".."}),
	))
	if err != nil {
		return nil, errors.Join(err, plugin.Close())
	}
	return &JpoetFacade{plugin: plugin, env: env}, nil
}

func (f *JpoetFacade) Evaluate(ctx context.Context, snippet string) (string, error) {
	return runWithContext(ctx, func() (string, error) {
		var out bytes.Buffer
		err := f.env.Eval(
			jpoet.EvalSnippetInput("main.jsonnet", snippet),
			jpoet.EvalWriterOutput(&out),
		)
		return out.String(), err
	})
}

func (f *JpoetFacade) Watch(ctx context.Context, snippet string) (Result, <-chan Result, func(), error) {
	type watchResult struct {
		initial     Result
		updates     <-chan Result
		unsubscribe func()
	}
	res, err := runWithContext(ctx, func() (watchResult, error) {
		var r watchResult
		unsubscribe, err := f.env.Watch(
			watch.WatchWithKey("main"),
			watch.WatchSnippetInput("main.jsonnet", snippet),
			watch.WatchValueOutput(&r.initial, &r.updates),
		)
		r.unsubscribe = unsubscribe
		return r, err
	})
	if res.unsubscribe == nil {
		res.unsubscribe = func() {}
	}
	return res.initial, res.updates, res.unsubscribe, err
}

func (f *JpoetFacade) Close() error {
	return errors.Join(f.plugin.Close(), f.env.Close())
}

func runWithContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}
	done := make(chan result, 1)
	go func() {
		value, err := fn()
		done <- result{value, err}
	}()
	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}
