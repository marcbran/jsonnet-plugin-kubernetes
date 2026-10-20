//go:build e2e

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const evaluationTimeout = 5 * time.Second

type Stage struct {
	t *testing.T

	facade  Facade
	tempDir string

	clusters []*fakeCluster

	watchUpdates     <-chan Result
	watchUnsubscribe func()

	LastOutput string
	LastError  string
}

func scenario(t *testing.T) (*Stage, *Stage, *Stage) {
	facade, err := NewE2EFacade()
	require.NoError(t, err)
	s := &Stage{
		t:       t,
		facade:  facade,
		tempDir: t.TempDir(),
	}
	t.Cleanup(s.close)
	return s, s, s
}

func (s *Stage) close() {
	if s.watchUnsubscribe != nil {
		s.watchUnsubscribe()
	}
	_ = s.facade.Close()
	for _, c := range s.clusters {
		c.server.CloseClientConnections()
		c.server.Close()
	}
}

func (s *Stage) and() *Stage {
	return s
}

func (s *Stage) the_jsonnet_is_evaluated(snippet string) *Stage {
	ctx, cancel := context.WithTimeout(context.Background(), evaluationTimeout)
	defer cancel()
	out, err := s.facade.Evaluate(ctx, snippet)
	s.record(out, err)
	return s
}

func (s *Stage) the_jsonnet_is_watched(snippet string) *Stage {
	ctx, cancel := context.WithTimeout(context.Background(), evaluationTimeout)
	defer cancel()
	initial, updates, unsubscribe, err := s.facade.Watch(ctx, snippet)
	s.watchUpdates = updates
	s.watchUnsubscribe = unsubscribe
	if err != nil {
		s.record("", err)
		return s
	}
	s.record(initial.Output, initial.Err)
	return s
}

func (s *Stage) record(out string, err error) {
	s.LastOutput = out
	s.LastError = ""
	if err != nil {
		s.LastError = err.Error()
	}
}

func (s *Stage) there_is_no_error() *Stage {
	require.Empty(s.t, s.LastError)
	return s
}

func (s *Stage) the_output_is(expected string) *Stage {
	assert.JSONEq(s.t, expected, s.LastOutput)
	return s
}

func (s *Stage) the_error_contains(expected string) *Stage {
	assert.Contains(s.t, s.LastError, expected)
	return s
}
