package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func TestResolveCommitCanvasAutoLayout(t *testing.T) {
	t.Run("not specified -> nil", func(t *testing.T) {
		assert.Nil(t, resolveCommitCanvasAutoLayout(false, nil))
	})

	t.Run("explicit disable -> nil", func(t *testing.T) {
		assert.Nil(t, resolveCommitCanvasAutoLayout(true, &pb.CanvasAutoLayout{}))
	})

	t.Run("explicit layout -> preserved", func(t *testing.T) {
		layout := &pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
			Scope:     pb.CanvasAutoLayout_SCOPE_FULL_CANVAS,
		}
		assert.Equal(t, layout, resolveCommitCanvasAutoLayout(true, layout))
	})
}

func TestReadRepositorySpecFileEmptyDraftIncludesNodeList(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	response, err := CreateCanvasVersion(ctx, r.Organization.ID.String(), canvas.ID.String(), "")
	require.NoError(t, err)

	versionID := response.GetVersion().GetMetadata().GetId()
	yamlText, err := ReadRepositorySpecFile(
		ctx,
		r.Organization.ID.String(),
		canvas.ID.String(),
		versionID,
		CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)
	require.Contains(t, yamlText, "nodes:")
	assert.True(t, strings.Contains(yamlText, "nodes: []") || strings.Contains(yamlText, "nodes:\n  []"))
}
