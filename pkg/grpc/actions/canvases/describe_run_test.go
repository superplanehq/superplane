package canvases

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("invalid run id -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		_, err := DescribeRun(context.Background(), r.Registry, canvas.ID, "not-a-uuid")
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("missing run -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

		_, err := DescribeRun(context.Background(), r.Registry, canvas.ID, uuid.New().String())
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	// Regression: a root event whose persisted JSON payload is not a top-level
	// object (e.g. raw scalar / array / null) must not surface as HTTP 500 from
	// DescribeRun. The gRPC `data` field is a Struct, but the serializer should
	// normalize unexpected payloads instead of returning a bare error.
	t.Run("root event with non-map data -> serializes successfully", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
			[]models.Edge{},
		)

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		run := createFinishedRun(t, rootEvent, models.CanvasRunResultPassed)

		// Overwrite the persisted payload AFTER the run has been finalized,
		// because createFinishedRun calls Save on the in-memory event and
		// would otherwise reset Data back to its default value.
		res := database.Conn().Exec(
			`UPDATE workflow_events SET data = '"raw-string-payload"'::jsonb WHERE id = ?`,
			rootEvent.ID,
		)
		require.NoError(t, res.Error)
		require.EqualValues(t, 1, res.RowsAffected)

		response, err := DescribeRun(context.Background(), r.Registry, canvas.ID, run.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.Run)
		require.NotNil(t, response.Run.RootEvent)
		require.Contains(t, response.Run.RootEvent.Data.AsMap(), "value")
		assert.Equal(t, "raw-string-payload", response.Run.RootEvent.Data.AsMap()["value"])
	})
}

// SerializeCanvasRun must not panic when the persisted row has nil time
// pointers. The columns themselves are NOT NULL today, but the model field
// types are *time.Time and historically have been dereferenced unconditionally,
// turning any divergence into a 500 instead of a clean nil-typed timestamp.
func Test__SerializeCanvasRun__HandlesNilTimestamps(t *testing.T) {
	canvasID := uuid.New()
	run := models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvasID,
		VersionID:  uuid.New(),
		State:      models.CanvasRunStateFinished,
		Result:     models.CanvasRunResultPassed,
		CreatedAt:  nil,
		UpdatedAt:  nil,
		FinishedAt: nil,
	}

	rootEventCreatedAt := time.Now()
	rootEvent := models.CanvasEvent{
		ID:         uuid.New(),
		WorkflowID: canvasID,
		NodeID:     "trigger",
		Channel:    "default",
		Data:       models.NewJSONValue(map[string]any{"k": "v"}),
		CreatedAt:  &rootEventCreatedAt,
	}

	serialized, err := SerializeCanvasRun(run, rootEvent, nil)
	require.NoError(t, err)
	require.NotNil(t, serialized)
	assert.Nil(t, serialized.CreatedAt)
	assert.Nil(t, serialized.UpdatedAt)
	assert.Nil(t, serialized.FinishedAt)
}
