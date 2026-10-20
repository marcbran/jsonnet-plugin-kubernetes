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
	"github.com/marcbran/jpoet/pkg/jpoet"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/streaming/pkg/httpstream"
)

const execTimeout = 30 * time.Second

type ExecInput struct {
	Context string
	Target  ExecTarget
	Command []string
}

type ExecTarget struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
}

type ExecOutput struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func Exec(clients *clientRegistry) jsonnet.NativeFunction {
	return jsonnet.NativeFunction{
		Name:   "exec",
		Params: ast.Identifiers{"ctx", "target", "command"},
		Func: func(args []any) (any, error) {
			ei, err := parseExecInput(args)
			if err != nil {
				return nil, err
			}
			out, err := runExec(context.Background(), clients, ei)
			if err != nil {
				return nil, err
			}
			return encodeOutput(out)
		},
	}
}

func ExecAction(clients *clientRegistry) jpoet.ActionFunc {
	return func(ctx context.Context, data map[string]any) (string, error) {
		args, err := actionArgs(data, "ctx", "target", "command")
		if err != nil {
			return "", err
		}
		ei, err := parseExecInput(args)
		if err != nil {
			return "", err
		}
		out, err := runExec(ctx, clients, ei)
		if err != nil {
			return "", err
		}
		return encodeActionOutput(out)
	}
}

func parseExecInput(args []any) (ExecInput, error) {
	if len(args) != 3 {
		return ExecInput{}, fmt.Errorf("expected ctx, target, and command")
	}
	contextName, ok := args[0].(string)
	if !ok {
		return ExecInput{}, fmt.Errorf("ctx must be a string")
	}
	target, err := decodeArg[ExecTarget](args[1], "target")
	if err != nil {
		return ExecInput{}, err
	}
	if target.Namespace == "" {
		return ExecInput{}, fmt.Errorf("target.namespace is required")
	}
	if target.Pod == "" {
		return ExecInput{}, fmt.Errorf("target.pod is required")
	}
	command, err := decodeArg[[]string](args[2], "command")
	if err != nil {
		return ExecInput{}, err
	}
	if len(command) == 0 {
		return ExecInput{}, fmt.Errorf("command must not be empty")
	}
	return ExecInput{Context: contextName, Target: target, Command: command}, nil
}

func runExec(ctx context.Context, clients *clientRegistry, ei ExecInput) (ExecOutput, error) {
	cl, err := clients.get(ei.Context)
	if err != nil {
		return ExecOutput{}, err
	}

	req := cl.core.Post().
		Namespace(ei.Target.Namespace).
		Resource("pods").
		Name(ei.Target.Pod).
		SubResource("exec").
		Param("stdout", "true").
		Param("stderr", "true")
	if ei.Target.Container != "" {
		req = req.Param("container", ei.Target.Container)
	}
	for _, c := range ei.Command {
		req = req.Param("command", c)
	}

	executor, err := newExecutor(cl, req.URL())
	if err != nil {
		return ExecOutput{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	var stdout, stderr bytes.Buffer
	exitCode := 0
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		var exitErr utilexec.ExitError
		if !errors.As(err, &exitErr) {
			return ExecOutput{}, err
		}
		exitCode = exitErr.ExitStatus()
	}

	return ExecOutput{ExitCode: exitCode, Stdout: stdout.String(), Stderr: stderr.String()}, nil
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
