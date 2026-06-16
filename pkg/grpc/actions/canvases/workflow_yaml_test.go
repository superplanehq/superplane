package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasYAMLFromVersionIncludesActionNodeType(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createGitCanvas(ctx, t, r, "action-node-type", []*componentpb.Node{
		{
			Id:        "wait-1",
			Name:      "Wait",
			Component: "wait",
			Configuration: structFromAnyMap(t, map[string]any{
				"mode":    "interval",
				"waitFor": "10",
				"unit":    "seconds",
			}),
		},
	})

	response, err := CreateCanvasVersion(ctx, r.GitProvider, r.Registry, r.Organization.ID.String(), canvasID, "")
	require.NoError(t, err)

	versionID := response.GetVersion().GetMetadata().GetId()
	yamlText, err := ReadRepositorySpecFile(
		ctx,
		r.Organization.ID.String(),
		canvasID,
		versionID,
		CanvasYAMLRepositoryPath,
	)
	require.NoError(t, err)
	require.Contains(t, yamlText, "component: wait")
	assert.True(t, strings.Contains(yamlText, "type: TYPE_ACTION") || strings.Contains(yamlText, "type: \"TYPE_ACTION\""))
}
