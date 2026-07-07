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
	"gorm.io/datatypes"
)

func TestCanvasYAMLFromVersionIncludesActionNodeType(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "wait-1",
			Name:   "Wait",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "wait"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"mode":    "interval",
				"waitFor": "10",
				"unit":    "seconds",
			}),
		},
	}, nil)

	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	yamlText, err := ReadRepositorySpecFile(ctx, canvas, liveVersion, CanvasYAMLRepositoryPath)
	require.NoError(t, err)
	require.Contains(t, yamlText, "component: wait")
	assert.True(t, strings.Contains(yamlText, "type: TYPE_ACTION") || strings.Contains(yamlText, "type: \"TYPE_ACTION\""))
}
