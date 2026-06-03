package repository

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func Test__renderRepositoryText(t *testing.T) {
	var output bytes.Buffer

	state := openapi_client.CANVASESCANVASREPOSITORYSTATE_STATE_READY
	repository := openapi_client.CanvasesCanvasRepository{
		Metadata: &openapi_client.CanvasesCanvasRepositoryMetadata{
			CanvasId:      stringPtr("canvas-1"),
			RepoId:        stringPtr("orgs/org-1/my-app"),
			Provider:      stringPtr("supergit"),
			Url:           stringPtr("https://app.superplane.com/git/550e8400-e29b-41d4-a716-446655440000.git"),
			DefaultBranch: stringPtr("main"),
		},
		Status: &openapi_client.CanvasesCanvasRepositoryStatus{
			State:   &state,
			HeadSha: stringPtr("abc123"),
		},
	}

	err := renderRepositoryText(&output, repository)
	require.NoError(t, err)
	require.Contains(t, output.String(), "Repository ID: orgs/org-1/canvases/canvas-1")
	require.Contains(t, output.String(), "Clone URL: https://app.superplane.com/git/550e8400-e29b-41d4-a716-446655440000.git")
	require.Contains(t, output.String(), "Head SHA: abc123")
}

func stringPtr(value string) *string {
	return &value
}
