package oidc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchExpectedClaims(t *testing.T) {
	t.Parallel()

	cmd := &verifyCommand{
		pipelineFile: strPtr(".semaphore/deploy.yml"),
		nodeID:       strPtr("deploy"),
	}

	err := matchExpectedClaims(map[string]any{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	}, cmd)
	require.NoError(t, err)

	cmd.canvasID = strPtr("expected-canvas")
	err = matchExpectedClaims(map[string]any{
		"canvas_id": "other-canvas",
	}, cmd)
	require.Error(t, err)
}

func strPtr(value string) *string {
	return &value
}
