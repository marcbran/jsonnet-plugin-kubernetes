//go:build e2e

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	kubernetespkg "github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/pkg/kubernetesopenapi"
)

type CLIFacade struct {
	binaryPath string
}

func NewCLIFacade() (kubernetespkg.Facade, error) {
	binaryPath := os.Getenv("KUBERNETES_OPENAPI_BINARY")
	if binaryPath == "" {
		return nil, fmt.Errorf("KUBERNETES_OPENAPI_BINARY is required")
	}
	_, err := os.Stat(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("unable to access KUBERNETES_OPENAPI_BINARY %q: %w", binaryPath, err)
	}
	return &CLIFacade{binaryPath: binaryPath}, nil
}

func (f *CLIFacade) FetchOpenAPI(ctx context.Context, in kubernetespkg.FetchOpenAPIInput) (kubernetespkg.FetchOpenAPIOutput, error) {
	args := []string{"fetch"}
	args = append(args, in.Contexts...)
	args = append(args, "--out", in.OutDir, "--format", "json", "-q")
	cmd := exec.CommandContext(ctx, f.binaryPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.String() != "" {
			return kubernetespkg.FetchOpenAPIOutput{}, errors.New(stderr.String())
		}
		return kubernetespkg.FetchOpenAPIOutput{}, err
	}
	var out kubernetespkg.FetchOpenAPIOutput
	err = json.Unmarshal(stdout.Bytes(), &out)
	if err != nil {
		return kubernetespkg.FetchOpenAPIOutput{}, err
	}
	return out, nil
}
