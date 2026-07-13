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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ListRuns__ReturnsRunsWithRootEventsAndExecutionRefs(t *testing.T) {
	r := support.Setup(t)
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

	response, err := ListRuns(context.Background(), r.Registry, canvas.ID, 0, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Runs, 1)

	serializedRun := response.Runs[0]
	assert.Equal(t, run.ID.String(), serializedRun.Id)
	assert.Equal(t, run.VersionID.String(), serializedRun.VersionId)
	assert.Equal(t, pb.CanvasRun_STATE_FINISHED, serializedRun.State)
	assert.Equal(t, pb.CanvasRun_RESULT_PASSED, serializedRun.Result)
	require.NotNil(t, serializedRun.RootEvent)
	assert.Equal(t, rootEvent.ID.String(), serializedRun.RootEvent.Id)
	require.Len(t, serializedRun.Executions, 1)
	assert.Equal(t, execution.ID.String(), serializedRun.Executions[0].Id)
	assert.Equal(t, uint32(1), response.TotalCount)
	assert.False(t, response.HasNextPage)
}

func Test__ListRuns__ReturnsRunsWithQueueItems(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Type: models.NodeTypeComponent},
			{NodeID: "node-2", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger", "default", nil, map[string]any{
		"review": "pending",
	})
	run := createStartedRun(t, rootEvent)
	queueItem := support.CreateQueueItem(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)

	emptyRootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	emptyRun := createStartedRun(t, emptyRootEvent)

	otherRootEvent := support.EmitCanvasEventForNode(t, otherCanvas.ID, "trigger", "default", nil)
	createStartedRun(t, otherRootEvent)
	support.CreateQueueItem(t, otherCanvas.ID, "trigger", otherRootEvent.ID, otherRootEvent.ID)

	response, err := ListRuns(context.Background(), r.Registry, canvas.ID, 0, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, response.Runs, 2)

	runsByID := map[string]*pb.CanvasRun{}
	for _, serializedRun := range response.Runs {
		runsByID[serializedRun.Id] = serializedRun
	}

	serializedRun := runsByID[run.ID.String()]
	require.NotNil(t, serializedRun)
	require.Len(t, serializedRun.QueueItems, 1)
	assert.Equal(t, queueItem.ID.String(), serializedRun.QueueItems[0].Id)
	assert.Equal(t, "node-1", serializedRun.QueueItems[0].NodeId)
	assert.NotNil(t, serializedRun.QueueItems[0].CreatedAt)
	require.NotNil(t, serializedRun.QueueItems[0].RootEvent)
	assert.Equal(t, rootEvent.ID.String(), serializedRun.QueueItems[0].RootEvent.Id)
	require.NotNil(t, serializedRun.QueueItems[0].Input)
	assert.Equal(t, "pending", serializedRun.QueueItems[0].Input.AsMap()["review"])

	require.NotNil(t, runsByID[emptyRun.ID.String()])
	assert.Empty(t, runsByID[emptyRun.ID.String()].QueueItems)
}

func Test__ListRuns__ScopesRunsToCanvas(t *testing.T) {
	r := support.Setup(t)
	canvasOne, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})
	canvasTwo, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

	rootEventOne := support.EmitCanvasEventForNode(t, canvasOne.ID, "trigger", "default", nil)
	rootEventTwo := support.EmitCanvasEventForNode(t, canvasTwo.ID, "trigger", "default", nil)
	runOne := createFinishedRun(t, rootEventOne, models.CanvasRunResultPassed)
	createFinishedRun(t, rootEventTwo, models.CanvasRunResultPassed)

	response, err := ListRuns(context.Background(), r.Registry, canvasOne.ID, 0, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, response.Runs, 1)
	assert.Equal(t, runOne.ID.String(), response.Runs[0].Id)
}

func Test__ListRuns__FiltersByStateOrResult(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

	startedRootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	startedRun := createStartedRun(t, startedRootEvent)

	failedRootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	failedRun := createFinishedRun(t, failedRootEvent, models.CanvasRunResultFailed)

	cancelledRootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	cancelledRun := createFinishedRun(t, cancelledRootEvent, models.CanvasRunResultCancelled)

	passedRootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	passedRun := createFinishedRun(t, passedRootEvent, models.CanvasRunResultPassed)

	// State + result filters are combined with OR so the status filter UI can ask for
	// "running OR passed" in a single request.
	response, err := ListRuns(
		context.Background(),
		r.Registry,
		canvas.ID,
		0,
		nil,
		[]pb.CanvasRun_State{pb.CanvasRun_STATE_STARTED},
		[]pb.CanvasRun_Result{pb.CanvasRun_RESULT_PASSED},
	)
	require.NoError(t, err)
	require.Len(t, response.Runs, 2)
	assert.Equal(t, uint32(2), response.TotalCount)
	assert.ElementsMatch(t,
		[]string{startedRun.ID.String(), passedRun.ID.String()},
		[]string{response.Runs[0].Id, response.Runs[1].Id},
	)

	// Result-only filter still narrows to the requested results.
	response, err = ListRuns(
		context.Background(),
		r.Registry,
		canvas.ID,
		0,
		nil,
		nil,
		[]pb.CanvasRun_Result{pb.CanvasRun_RESULT_FAILED, pb.CanvasRun_RESULT_CANCELLED},
	)
	require.NoError(t, err)
	require.Len(t, response.Runs, 2)
	assert.ElementsMatch(t,
		[]string{failedRun.ID.String(), cancelledRun.ID.String()},
		[]string{response.Runs[0].Id, response.Runs[1].Id},
	)

	// State-only filter narrows to running runs.
	response, err = ListRuns(
		context.Background(),
		r.Registry,
		canvas.ID,
		0,
		nil,
		[]pb.CanvasRun_State{pb.CanvasRun_STATE_STARTED},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, response.Runs, 1)
	assert.Equal(t, startedRun.ID.String(), response.Runs[0].Id)
	assert.Equal(t, uint32(1), response.TotalCount)
}

func Test__ListRuns__RejectsUnknownFilterValues(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

	_, err := ListRuns(
		context.Background(),
		r.Registry,
		canvas.ID,
		0,
		nil,
		[]pb.CanvasRun_State{pb.CanvasRun_STATE_UNKNOWN},
		nil,
	)
	require.Error(t, err)

	_, err = ListRuns(
		context.Background(),
		r.Registry,
		canvas.ID,
		0,
		nil,
		nil,
		[]pb.CanvasRun_Result{pb.CanvasRun_RESULT_UNKNOWN},
	)
	require.Error(t, err)
}

func createStartedRun(t *testing.T, rootEvent *models.CanvasEvent) *models.CanvasRun {
	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		return err
	}))

	return run
}

func createFinishedRun(t *testing.T, rootEvent *models.CanvasEvent, result string) *models.CanvasRun {
	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}

		if err := rootEvent.RoutedInTransaction(tx); err != nil {
			return err
		}

		now := time.Now()
		return tx.Model(run).Updates(map[string]any{
			"state":       models.CanvasRunStateFinished,
			"result":      result,
			"updated_at":  &now,
			"finished_at": &now,
		}).Error
	}))

	return run
}

func createRunExecution(t *testing.T, run *models.CanvasRun, rootEventID uuid.UUID, nodeID string, result string) *models.CanvasNodeExecution {
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       rootEventID,
		State:         models.CanvasNodeExecutionStateFinished,
		Result:        result,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}
