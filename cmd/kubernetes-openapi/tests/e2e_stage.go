//go:build e2e

package tests

import (
	"testing"

	kubernetespkg "github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/pkg/kubernetesopenapi"
	"github.com/stretchr/testify/require"
)

type Stage struct {
	t require.TestingT

	facade kubernetespkg.Facade

	tempDir  string
	outDir   string
	contexts []string

	kubeconfigClusters []kubeconfigCluster

	lastOutput kubernetespkg.FetchOpenAPIOutput
	lastErr    string
}

type kubeconfigCluster struct {
	Context string
	Server  string
}

func scenario(t *testing.T) (*Stage, *Stage, *Stage) {
	facade, err := NewCLIFacade()
	require.NoError(t, err)
	tempDir := t.TempDir()
	s := &Stage{
		t:       t,
		facade:  facade,
		tempDir: tempDir,
		outDir:  tempDir,
	}
	return s, s, s
}

func (s *Stage) and() *Stage {
	return s
}
