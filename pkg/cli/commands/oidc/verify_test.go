package oidc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExpectedClaims(t *testing.T) {
	t.Parallel()

	expected, err := parseExpectedClaims([]string{
		"pipeline_file=.semaphore/deploy.yml",
		"node_id=deploy",
	})
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	}, expected)

	_, err = parseExpectedClaims([]string{"invalid"})
	require.Error(t, err)
}

func TestMatchExpectedClaims(t *testing.T) {
	t.Parallel()

	err := matchExpectedClaims(map[string]any{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	}, map[string]string{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	})
	require.NoError(t, err)

	err = matchExpectedClaims(map[string]any{
		"canvas_id": "other-canvas",
	}, map[string]string{
		"canvas_id": "expected-canvas",
	})
	require.Error(t, err)
}
