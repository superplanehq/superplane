package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__DescribeRun(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns run with root event and execution refs", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{NodeID: "trigger", Type: models.NodeTypeTrigger},
				{NodeID: "node-1", Type: models.NodeTypeComponent},
			},
			[]models.Edge{},
		)

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		run := createFinishedRun(t, rootEvent, models.CanvasRunResultPassed)
		execution := createRunExecution(t, run, rootEvent.ID, "node-1", models.CanvasNodeExecutionResultPassed)

		response, err := DescribeRun(context.Background(), r.Registry, canvas.ID, run.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Run)

		serializedRun := response.Run
		assert.Equal(t, run.ID.String(), serializedRun.Id)
		assert.Equal(t, run.VersionID.String(), serializedRun.VersionId)
		assert.Equal(t, pb.CanvasRun_STATE_FINISHED, serializedRun.State)
		assert.Equal(t, pb.CanvasRun_RESULT_PASSED, serializedRun.Result)
		require.NotNil(t, serializedRun.RootEvent)
		assert.Equal(t, rootEvent.ID.String(), serializedRun.RootEvent.Id)
		require.Len(t, serializedRun.Executions, 1)
		assert.Equal(t, execution.ID.String(), serializedRun.Executions[0].Id)
	})

	t.Run("scopes run to canvas", func(t *testing.T) {
		canvasOne, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})
		canvasTwo, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		rootEventTwo := support.EmitCanvasEventForNode(t, canvasTwo.ID, "trigger", "default", nil)
		runTwo := createFinishedRun(t, rootEventTwo, models.CanvasRunResultPassed)

		_, err := DescribeRun(context.Background(), r.Registry, canvasOne.ID, runTwo.ID.String())
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("invalid run id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		_, err := DescribeRun(context.Background(), r.Registry, canvas.ID, "not-a-uuid")
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("missing run -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		_, err := DescribeRun(context.Background(), r.Registry, canvas.ID, uuid.New().String())
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})
}
