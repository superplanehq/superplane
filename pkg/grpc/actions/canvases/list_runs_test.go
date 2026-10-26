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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ListRuns__SearchesRootTitlesAndTriggersBeforePagination(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	nodes := []models.CanvasNode{
		{NodeID: "trigger-a", Type: models.NodeTypeTrigger},
		{NodeID: "trigger-b", Type: models.NodeTypeTrigger},
	}
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, nil)
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, nil)
	now := time.Now()
	createRun := func(canvas *models.Canvas, trigger, title, result string, age time.Duration) *models.CanvasRun {
		event := support.EmitCanvasEventForNode(t, canvas.ID, trigger, "default", nil)
		require.NoError(t, db.Model(event).Update("custom_name", title).Error)
		run := createFinishedRun(t, event, result)
		require.NoError(t, db.Model(run).Update("created_at", now.Add(-age)).Error)
		return run
	}
	older := createRun(canvas, "trigger-a", "!12 - Deploy API", models.CanvasRunResultPassed, 3*time.Minute)
	newer := createRun(canvas, "trigger-a", "!13 - DEPLOY worker", models.CanvasRunResultPassed, 2*time.Minute)
	failed := createRun(canvas, "trigger-b", "!14 - Deploy UI", models.CanvasRunResultFailed, time.Minute)
	literal := createRun(canvas, "trigger-a", "Fix 100%_coverage", models.CanvasRunResultPassed, 0)
	createRun(otherCanvas, "trigger-a", "Deploy another canvas", models.CanvasRunResultPassed, 0)
	unnamed := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-a", "default", nil)
	createFinishedRun(t, unnamed, models.CanvasRunResultPassed)

	for _, test := range []struct {
		name    string
		request *pb.ListRunsRequest
		want    []string
	}{
		{"title substring ignores case and surrounding whitespace", &pb.ListRunsRequest{Search: "  dEpLoY  "}, []string{older.ID.String(), newer.ID.String(), failed.ID.String()}},
		{"trigger only", &pb.ListRunsRequest{TriggerIds: []string{"trigger-b"}}, []string{failed.ID.String()}},
		{"multiple triggers", &pb.ListRunsRequest{Search: "deploy", TriggerIds: []string{"trigger-a", "trigger-b"}}, []string{older.ID.String(), newer.ID.String(), failed.ID.String()}},
		{"combined status trigger and title", &pb.ListRunsRequest{Search: "deploy", TriggerIds: []string{"trigger-a"}, Results: []pb.CanvasRun_Result{pb.CanvasRun_RESULT_PASSED}}, []string{older.ID.String(), newer.ID.String()}},
		{"wildcards are literal", &pb.ListRunsRequest{Search: "%_"}, []string{literal.ID.String()}},
		{"no matches", &pb.ListRunsRequest{Search: "absent title"}, nil},
		{"unknown trigger", &pb.ListRunsRequest{TriggerIds: []string{"absent-trigger"}}, nil},
		{"run IDs are not titles", &pb.ListRunsRequest{Search: older.ID.String()}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := ListRuns(t.Context(), db, canvas, test.request)
			require.NoError(t, err)
			ids := make([]string, 0, len(response.Runs))
			for _, run := range response.Runs {
				ids = append(ids, run.Id)
			}
			assert.ElementsMatch(t, test.want, ids)
			assert.EqualValues(t, len(test.want), response.TotalCount)
			assert.False(t, response.HasNextPage)
		})
	}

	first, err := ListRuns(t.Context(), db, canvas, &pb.ListRunsRequest{Search: "deploy", TriggerIds: []string{"trigger-a"}, Limit: 1})
	require.NoError(t, err)
	require.Len(t, first.Runs, 1)
	assert.Equal(t, newer.ID.String(), first.Runs[0].Id)
	assert.EqualValues(t, 2, first.TotalCount)
	assert.True(t, first.HasNextPage)
	second, err := ListRuns(t.Context(), db, canvas, &pb.ListRunsRequest{Search: "deploy", TriggerIds: []string{"trigger-a"}, Limit: 1, Before: timestamppb.New(first.LastTimestamp.AsTime())})
	require.NoError(t, err)
	require.Len(t, second.Runs, 1)
	assert.Equal(t, older.ID.String(), second.Runs[0].Id)
	assert.EqualValues(t, 2, second.TotalCount)
}

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

	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{})
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

func Test__ListRuns__ReturnsExistingTaskDecision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger},
		{NodeID: "create", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "createWorkOrder"}})},
	}, nil)
	root := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createFinishedRun(t, root, models.CanvasRunResultPassed)
	execution := createRunExecution(t, run, root.ID, "create", models.CanvasNodeExecutionResultPassed)
	taskID := uuid.New().String()
	require.NoError(t, db.Model(&execution).Update("metadata", datatypes.NewJSONType(map[string]any{"existing_task_id": taskID, "unrelated": "private execution data"})).Error)
	response, err := ListRuns(t.Context(), db, canvas, &pb.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, response.Runs, 1)
	require.Len(t, response.Runs[0].Executions, 1)
	assert.Equal(t, taskID, response.Runs[0].Executions[0].ExistingTaskId)
	detail, err := DescribeRun(t.Context(), db, canvas, run.ID.String(), false)
	require.NoError(t, err)
	assert.Equal(t, taskID, detail.Run.Executions[0].ExistingTaskId)
}

// A passed filter can emit on either branch. Lists must expose that evidence
// without loading output payloads or interpreting the current canvas version.
func Test__ListRuns__ReturnsRecordedChannelsAndHistoricalComponents(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger},
		{NodeID: "filter", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "if"}})},
	}, nil)
	root := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	rejected := createFinishedRun(t, root, models.CanvasRunResultPassed)
	rejectedExecution := createRunExecution(t, rejected, root.ID, "filter", models.CanvasNodeExecutionResultPassed)
	support.EmitCanvasEventForNode(t, canvas.ID, "filter", "false", &rejectedExecution.ID)
	support.EmitCanvasEventForNode(t, canvas.ID, "filter", "false", &rejectedExecution.ID)
	nextRoot := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	accepted := createFinishedRun(t, nextRoot, models.CanvasRunResultPassed)
	acceptedExecution := createRunExecution(t, accepted, nextRoot.ID, "filter", models.CanvasNodeExecutionResultPassed)
	support.EmitCanvasEventForNode(t, canvas.ID, "filter", "true", &acceptedExecution.ID)
	// Later canvas edits must not change the interpretation of recorded runs.
	require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ? AND node_id = ?", canvas.ID, "filter").Update("ref", datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "other"}})).Error)
	newVersion := models.CanvasVersion{
		ID: uuid.New(), WorkflowID: canvas.ID,
		Nodes: datatypes.NewJSONSlice([]models.Node{{ID: "filter", Type: models.NodeTypeComponent, Ref: models.NodeRef{Component: &models.ComponentRef{Name: "other"}}}}),
		Edges: datatypes.NewJSONSlice([]models.Edge{}),
	}
	require.NoError(t, db.Create(&newVersion).Error)
	require.NoError(t, db.Model(canvas).Update("live_version_id", newVersion.ID).Error)
	response, err := ListRuns(t.Context(), db, canvas, &pb.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, response.Runs, 2)
	channels := map[string][]string{}
	for _, run := range response.Runs {
		require.Len(t, run.Executions, 1)
		assert.Equal(t, "if", run.Executions[0].ComponentName)
		channels[run.Id] = run.Executions[0].OutputChannels
	}
	assert.Equal(t, []string{"false"}, channels[rejected.ID.String()])
	assert.Equal(t, []string{"true"}, channels[accepted.ID.String()])
	detail, err := DescribeRun(t.Context(), db, canvas, rejected.ID.String(), false)
	require.NoError(t, err)
	assert.Equal(t, []string{"false"}, detail.Run.Executions[0].OutputChannels)
	assert.Equal(t, "if", detail.Run.Executions[0].ComponentName)
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

	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{})
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

	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvasOne, &pb.ListRunsRequest{})
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
	response, err := ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{States: []pb.CanvasRun_State{pb.CanvasRun_STATE_STARTED}, Results: []pb.CanvasRun_Result{pb.CanvasRun_RESULT_PASSED}})
	require.NoError(t, err)
	require.Len(t, response.Runs, 2)
	assert.Equal(t, uint32(2), response.TotalCount)
	assert.ElementsMatch(t,
		[]string{startedRun.ID.String(), passedRun.ID.String()},
		[]string{response.Runs[0].Id, response.Runs[1].Id},
	)

	// Result-only filter still narrows to the requested results.
	response, err = ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{Results: []pb.CanvasRun_Result{pb.CanvasRun_RESULT_FAILED, pb.CanvasRun_RESULT_CANCELLED}})
	require.NoError(t, err)
	require.Len(t, response.Runs, 2)
	assert.ElementsMatch(t,
		[]string{failedRun.ID.String(), cancelledRun.ID.String()},
		[]string{response.Runs[0].Id, response.Runs[1].Id},
	)

	// State-only filter narrows to running runs.
	response, err = ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{States: []pb.CanvasRun_State{pb.CanvasRun_STATE_STARTED}})
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

	_, err := ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{States: []pb.CanvasRun_State{pb.CanvasRun_STATE_UNKNOWN}})
	require.Error(t, err)

	_, err = ListRuns(context.Background(), database.DB(t.Context()), canvas, &pb.ListRunsRequest{Results: []pb.CanvasRun_Result{pb.CanvasRun_RESULT_UNKNOWN}})
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

	childResponse, err := ListRuns(context.Background(), database.DB(t.Context()), childCanvas, &pb.ListRunsRequest{
		States:     []pb.CanvasRun_State{pb.CanvasRun_STATE_PENDING},
		TriggerIds: []string{"onRun"},
	})
	require.NoError(t, err)
	require.Len(t, childResponse.Runs, 1)
	assert.Equal(t, childRun.ID.String(), childResponse.Runs[0].Id)
	assert.Equal(t, "onRun", childResponse.Runs[0].GetNodeId())
	assert.EqualValues(t, 1, childResponse.TotalCount)
	assert.Equal(t, pb.CanvasRun_STATE_PENDING, childResponse.Runs[0].State)
	assert.Nil(t, childResponse.Runs[0].RootEvent)

	parentResponse, err := ListRuns(context.Background(), database.DB(t.Context()), parentCanvas, &pb.ListRunsRequest{})
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

	parentResponse, err := ListRuns(context.Background(), database.DB(t.Context()), parentCanvas, &pb.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, parentResponse.Runs, 1)
	require.Len(t, parentResponse.Runs[0].Executions, 1)
	require.Len(t, parentResponse.Runs[0].Executions[0].Runs, 1)
	assert.Equal(t, childRun.ID.String(), parentResponse.Runs[0].Executions[0].Runs[0].Id)
	assert.Equal(t, childCanvas.ID.String(), parentResponse.Runs[0].Executions[0].Runs[0].CanvasId)

	childResponse, err := DescribeRun(context.Background(), database.DB(t.Context()), childCanvas, childRun.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, childResponse.Run.Parent)
	assert.Equal(t, parentRun.ID.String(), childResponse.Run.Parent.Id)
	assert.Equal(t, parentCanvas.ID.String(), childResponse.Run.Parent.CanvasId)
}

func Test__ListRuns__IncludesUsageTotalsAndModels(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "score", "start")
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, entry, "default", nil, map[string]any{
		"workOrder": map[string]any{"id": uuid.New().String()},
	})
	run := createFinishedRun(t, rootEvent, models.CanvasRunResultPassed)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		FundingSource:   models.UsageFundingSourceHosted,
	}))

	response, err := ListRuns(context.Background(), db, canvas, &pb.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, response.Runs, 1)
	assert.Equal(t, run.ID.String(), response.Runs[0].GetId())
	assert.EqualValues(t, 1_000_000, response.Runs[0].GetTotalTokens())
	assert.Positive(t, response.Runs[0].GetCostCents())
	assert.Equal(t, []string{"anthropic/claude-sonnet-4-6"}, response.Runs[0].GetModels())
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
