package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestReadRepositorySpecFileEmptyLiveIncludesNodeList(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	yamlText, err := ReadRepositorySpecFile(ctx, canvas, liveVersion, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	require.Contains(t, yamlText, "nodes:")
	assert.True(t, strings.Contains(yamlText, "nodes: []") || strings.Contains(yamlText, "nodes:\n  []"))
}
