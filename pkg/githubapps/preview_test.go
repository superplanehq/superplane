package githubapps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func TestBuildPreviewUsesCanvasMetadata(t *testing.T) {
	preview, err := BuildPreview("github.com/superplanehq/preview-env-github-digitalocean")
	require.NoError(t, err)

	assert.Equal(t, "Preview Environments - storejs complex", preview.CanvasName)
	assert.Equal(t, "StoreJS preview", preview.Description)
	assert.Equal(t, "Install Preview Environments - storejs complex", preview.Title)
	assert.Equal(t, "Preview Environments - storejs complex", preview.DefaultName)
}

func TestPreviewFromCanvasFallsBackToRepoNameWhenCanvasNameMissing(t *testing.T) {
	preview := previewFromCanvas(
		&Repository{Owner: "acme", Name: "preview-env-github-digitalocean"},
		&pb.Canvas{Metadata: &pb.Canvas_Metadata{Description: "A preview app"}},
		"main",
	)

	assert.Equal(t, "Install Preview Env Github Digitalocean", preview.Title)
	assert.Equal(t, "Preview Env Github Digitalocean", preview.DefaultName)
	assert.Equal(t, "A preview app", preview.Description)
	assert.Empty(t, preview.CanvasName)
}

func TestPreviewFromCanvasUsesCanvasMetadata(t *testing.T) {
	preview := previewFromCanvas(
		&Repository{Owner: "acme", Name: "ignored-repo"},
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Name:        "My Workflow",
				Description: "Does useful things",
			},
		},
		"main",
	)

	assert.Equal(t, "Install My Workflow", preview.Title)
	assert.Equal(t, "My Workflow", preview.DefaultName)
	assert.Equal(t, "Does useful things", preview.Description)
	assert.Equal(t, "My Workflow", preview.CanvasName)
}
