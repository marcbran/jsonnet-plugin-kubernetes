package kubernetes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/streaming/pkg/httpstream"
)

const execTimeout = 30 * time.Second

func Exec(clients *clientRegistry) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "exec",
		Params: ast.Identifiers{"ctx", "target", "command"},
		Func: func(args []any) (any, error) {
			return doExec(clients, args)
		},
	}
}

func doExec(clients *clientRegistry, args []any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("expected ctx, target, and command")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("ctx must be a string")
	}
	target, err := decodeArg[execTarget](args[1], "target")
	if err != nil {
		return nil, err
	}
	if err := target.validate(); err != nil {
		return nil, err
	}
	command, err := decodeArg[[]string](args[2], "command")
	if err != nil {
		return nil, err
	}
	if len(command) == 0 {
		return nil, fmt.Errorf("command must not be empty")
	}

	cl, err := clients.get(contextName)
	if err != nil {
		return nil, err
	}

	req := cl.core.Post().
		Namespace(target.Namespace).
		Resource("pods").
		Name(target.Pod).
		SubResource("exec").
		Param("stdout", "true").
		Param("stderr", "true")
	if target.Container != "" {
		req = req.Param("container", target.Container)
	}
	for _, c := range command {
		req = req.Param("command", c)
	}

	executor, err := newExecutor(cl, req.URL())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	exitCode := 0
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		var exitErr utilexec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, err
		}
		exitCode = exitErr.ExitStatus()
	}

	return map[string]any{
		"stdout":   stdout.String(),
		"stderr":   stderr.String(),
		"exitCode": float64(exitCode),
	}, nil
}

func newExecutor(cl *clients, u *url.URL) (remotecommand.Executor, error) {
	websocket, err := remotecommand.NewWebSocketExecutor(cl.config, http.MethodGet, u.String())
	if err != nil {
		return nil, err
	}
	spdy, err := remotecommand.NewSPDYExecutor(cl.config, http.MethodPost, u)
	if err != nil {
		return nil, err
	}
	return remotecommand.NewFallbackExecutor(websocket, spdy, func(err error) bool {
		return httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err)
	})
}

type execTarget struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
}

func (t execTarget) validate() error {
	if t.Namespace == "" {
		return fmt.Errorf("target.namespace is required")
	}
	if t.Pod == "" {
		return fmt.Errorf("target.pod is required")
	}
	return nil
}
