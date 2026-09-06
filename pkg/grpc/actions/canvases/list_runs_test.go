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

	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvas, 0, nil, nil, nil)
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

	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvas, 0, nil, nil, nil)
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

	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvasOne, 0, nil, nil, nil)
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
		database.DB(t.Context()),
		canvas,
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
		database.DB(t.Context()),
		canvas,
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
		database.DB(t.Context()),
		canvas,
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

func Test__RunStateMapping__Pending(t *testing.T) {
	assert.Equal(t, pb.CanvasRun_STATE_PENDING, RunStateToProto(models.CanvasRunStatePending))

	modelState, err := ProtoRunStateToModel(pb.CanvasRun_STATE_PENDING)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStatePending, modelState)
}

func Test__ListRuns__RejectsUnknownFilterValues(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, []models.Edge{})

	_, err := ListRuns(
		context.Background(),
		database.DB(t.Context()),
		canvas,
		0,
		nil,
		[]pb.CanvasRun_State{pb.CanvasRun_STATE_UNKNOWN},
		nil,
	)
	require.Error(t, err)

	_, err = ListRuns(
		context.Background(),
		database.DB(t.Context()),
		canvas,
		0,
		nil,
		nil,
		[]pb.CanvasRun_Result{pb.CanvasRun_RESULT_UNKNOWN},
	)
	require.Error(t, err)
}

func Test__ListRuns__ReturnsPendingSubRunWithoutRootEvent(t *testing.T) {
	r := support.Setup(t)

	parentCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "runApp", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)
	childCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	parentRootEvent := support.EmitCanvasEventForNode(t, parentCanvas.ID, "trigger", "default", nil)
	parentRun := createStartedRun(t, parentRootEvent)
	parentExecution := createRunExecution(t, parentRun, parentRootEvent.ID, "runApp", models.CanvasNodeExecutionResultPassed)

	childRun := createSubRunRecord(
		t,
		childCanvas.ID,
		"onRun",
		&parentRun.ID,
		&parentCanvas.ID,
		&parentExecution.ID,
		models.CanvasRunStatePending,
		"",
	)

	childResponse, err := ListRuns(
		context.Background(),
		database.DB(t.Context()),
		childCanvas,
		0,
		nil,
		[]pb.CanvasRun_State{pb.CanvasRun_STATE_PENDING},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, childResponse.Runs, 1)
	assert.Equal(t, childRun.ID.String(), childResponse.Runs[0].Id)
	assert.Equal(t, pb.CanvasRun_STATE_PENDING, childResponse.Runs[0].State)
	assert.Nil(t, childResponse.Runs[0].RootEvent)

	parentResponse, err := ListRuns(context.Background(), database.DB(t.Context()), parentCanvas, 0, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, parentResponse.Runs, 1)
	require.Len(t, parentResponse.Runs[0].Executions, 1)
	require.Len(t, parentResponse.Runs[0].Executions[0].Runs, 1)
	assert.Equal(t, childRun.ID.String(), parentResponse.Runs[0].Executions[0].Runs[0].Id)
	assert.Equal(t, pb.CanvasRun_STATE_PENDING, parentResponse.Runs[0].Executions[0].Runs[0].State)
}

func Test__ListRuns__ReturnsSubRunRelationshipRefs(t *testing.T) {
	r := support.Setup(t)

	parentCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "runApp", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)
	childCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		[]models.Edge{},
	)

	parentRootEvent := support.EmitCanvasEventForNode(t, parentCanvas.ID, "trigger", "default", nil)
	parentRun := createStartedRun(t, parentRootEvent)
	parentExecution := createRunExecution(t, parentRun, parentRootEvent.ID, "runApp", models.CanvasNodeExecutionResultPassed)

	childRun := createSubRunRecord(
		t,
		childCanvas.ID,
		"onRun",
		&parentRun.ID,
		&parentCanvas.ID,
		&parentExecution.ID,
		models.CanvasRunStateStarted,
		models.CanvasRunResultPassed,
	)
	childRootEvent := support.EmitCanvasEventForNode(t, childCanvas.ID, "onRun", "default", nil)
	require.NoError(t, database.Conn().Model(&childRootEvent).Update("run_id", childRun.ID).Error)

	parentResponse, err := ListRuns(context.Background(), database.DB(t.Context()), parentCanvas, 0, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, parentResponse.Runs, 1)
	require.Len(t, parentResponse.Runs[0].Executions, 1)
	require.Len(t, parentResponse.Runs[0].Executions[0].Runs, 1)
	assert.Equal(t, childRun.ID.String(), parentResponse.Runs[0].Executions[0].Runs[0].Id)
	assert.Equal(t, childCanvas.ID.String(), parentResponse.Runs[0].Executions[0].Runs[0].CanvasId)

	childResponse, err := DescribeRun(context.Background(), database.DB(t.Context()), childCanvas, childRun.ID.String())
	require.NoError(t, err)
	require.NotNil(t, childResponse.Run.Parent)
	assert.Equal(t, parentRun.ID.String(), childResponse.Run.Parent.Id)
	assert.Equal(t, parentCanvas.ID.String(), childResponse.Run.Parent.CanvasId)
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

func createSubRunRecord(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	parentRunID *uuid.UUID,
	parentWorkflowID *uuid.UUID,
	parentExecutionID *uuid.UUID,
	state string,
	result string,
) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:                uuid.New(),
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		VersionID:         liveVersion.ID,
		ParentRunID:       parentRunID,
		ParentWorkflowID:  parentWorkflowID,
		ParentExecutionID: parentExecutionID,
		State:             state,
		Result:            result,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createOpenRun(t *testing.T, rootEvent *models.CanvasEvent) *models.CanvasRun {
	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}

		return rootEvent.RoutedInTransaction(tx)
	}))

	return run
}

func createActiveExecutionInQueue(t *testing.T, run *models.CanvasRun, rootEventID uuid.UUID, nodeID, queueName string) *models.CanvasNodeExecution {
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       rootEventID,
		State:         models.CanvasNodeExecutionStateStarted,
		QueueName:     &queueName,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func Test__SerializeCanvasRun__ScopesBlockingExecutionsPerNode(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createOpenRun(t, rootEvent)

	node1Execution := createActiveExecutionInQueue(t, run, rootEvent.ID, "node-1", "shared")
	node2Execution := createActiveExecutionInQueue(t, run, rootEvent.ID, "node-2", "shared")

	node1QueueItem := createNodeQueueItem(t, canvas.ID, "node-1", rootEvent.ID, nil)
	require.NoError(t, database.Conn().Model(node1QueueItem).Update("queue_name", "shared").Error)
	node1QueueItem.QueueName = ptr("shared")

	node2QueueItem := createNodeQueueItem(t, canvas.ID, "node-2", rootEvent.ID, nil)
	require.NoError(t, database.Conn().Model(node2QueueItem).Update("queue_name", "shared").Error)
	node2QueueItem.QueueName = ptr("shared")

	serializedRun, err := SerializeCanvasRun(
		database.Conn(),
		*run,
		*rootEvent,
		[]models.CanvasNodeExecution{*node1Execution, *node2Execution},
		[]models.CanvasNodeQueueItem{*node1QueueItem, *node2QueueItem},
		nil,
		map[string][]models.CanvasRun{},
	)
	require.NoError(t, err)
	require.Len(t, serializedRun.QueueItems, 2)

	byNodeID := map[string]*pb.CanvasNodeQueueItem{}
	for _, item := range serializedRun.QueueItems {
		byNodeID[item.NodeId] = item
	}

	require.Len(t, byNodeID["node-1"].BlockingExecutions, 1)
	assert.Equal(t, node1Execution.ID.String(), byNodeID["node-1"].BlockingExecutions[0].Id)

	require.Len(t, byNodeID["node-2"].BlockingExecutions, 1)
	assert.Equal(t, node2Execution.ID.String(), byNodeID["node-2"].BlockingExecutions[0].Id)
}

func Test__SerializeCanvasRuns__BatchesBlockingExecutionsAcrossRuns(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent1 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run1 := createOpenRun(t, rootEvent1)
	execution1 := createActiveExecutionInQueue(t, run1, rootEvent1.ID, "node-1", "node-1")
	queueItem1 := createNodeQueueItem(t, canvas.ID, "node-1", rootEvent1.ID, nil)
	require.NoError(t, database.Conn().Model(queueItem1).Update("queue_name", "node-1").Error)
	queueItem1.QueueName = ptr("node-1")

	rootEvent2 := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run2 := createOpenRun(t, rootEvent2)
	queueItem2 := createNodeQueueItem(t, canvas.ID, "node-1", rootEvent2.ID, nil)
	require.NoError(t, database.Conn().Model(queueItem2).Update("queue_name", "node-1").Error)
	queueItem2.QueueName = ptr("node-1")

	serialized, err := SerializeCanvasRuns(
		database.Conn(),
		[]models.CanvasRun{*run1, *run2},
		map[string]models.CanvasEvent{
			run1.ID.String(): *rootEvent1,
			run2.ID.String(): *rootEvent2,
		},
		map[string][]models.CanvasNodeExecution{
			run1.ID.String(): {*execution1},
		},
		map[string][]models.CanvasNodeQueueItem{
			run1.ID.String(): {*queueItem1},
			run2.ID.String(): {*queueItem2},
		},
		map[string]models.CanvasRun{},
		map[string][]models.CanvasRun{},
	)
	require.NoError(t, err)
	require.Len(t, serialized, 2)

	serializedByRunID := map[string]*pb.CanvasRun{}
	for _, run := range serialized {
		serializedByRunID[run.Id] = run
	}

	require.Len(t, serializedByRunID[run1.ID.String()].QueueItems, 1)
	require.Len(t, serializedByRunID[run1.ID.String()].QueueItems[0].BlockingExecutions, 1)
	assert.Equal(t, execution1.ID.String(), serializedByRunID[run1.ID.String()].QueueItems[0].BlockingExecutions[0].Id)

	require.Len(t, serializedByRunID[run2.ID.String()].QueueItems, 1)
	require.Len(t, serializedByRunID[run2.ID.String()].QueueItems[0].BlockingExecutions, 1)
	assert.Equal(t, execution1.ID.String(), serializedByRunID[run2.ID.String()].QueueItems[0].BlockingExecutions[0].Id)
}
