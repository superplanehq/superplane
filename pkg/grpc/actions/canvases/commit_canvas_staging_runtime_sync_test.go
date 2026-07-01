package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
)

func TestCommitCanvasStagingSyncsRuntimeNodesOnMain(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	require.NoError(t, database.Conn().Where("workflow_id = ?", canvas.ID).Delete(&models.CanvasNode{}).Error)

	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	canvasYAML := `apiVersion: v1
kind: Canvas
metadata:
  name: ` + canvas.Name + `
spec:
  nodes:
    - id: start-node
      name: start-node
      type: TYPE_TRIGGER
      component: start
      configuration:
        templates:
          - name: Hello World
            payload:
              message: Hello, World!
  edges: []
`

	_, err = StageRepositorySpecFileOperations(
		ctx,
		orgID,
		canvas.ID.String(),
		liveVersion.ID.String(),
		models.CanvasGitBranchMain,
		[]*pb.CanvasRepositoryFileOperation{
			{Path: CanvasYAMLRepositoryPath, Content: []byte(canvasYAML)},
		},
	)
	require.NoError(t, err)

	_, err = CommitCanvasStaging(
		ctx,
		nil,
		nil,
		r.Encryptor,
		r.Registry,
		orgID,
		canvas.ID.String(),
		liveVersion.ID.String(),
		models.CanvasGitBranchMain,
		"Add start trigger",
		"",
		testWebhookBaseURL,
		r.AuthService,
	)
	require.NoError(t, err)

	runtimeNodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	require.Len(t, runtimeNodes, 1)
	assert.Equal(t, "start-node", runtimeNodes[0].NodeID)

	_, err = InvokeNodeTriggerHook(
		ctx,
		r.AuthService,
		r.Encryptor,
		r.Registry,
		r.Organization.ID,
		canvas.ID,
		"start-node",
		"run",
		map[string]any{"template": "Hello World"},
		testWebhookBaseURL,
	)
	require.NoError(t, err)
}
