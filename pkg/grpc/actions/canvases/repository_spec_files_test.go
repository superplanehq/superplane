package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/test/support"
)

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
