package canvases

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__ListRuns__IncludeTaskReturnsLinkedTaskLabels(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "RUNT")
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	canvas.FactoryID = &factory.ID

	executionOrder := createTaskOrder(t, db, factory, "Line task")
	require.NoError(t, db.Model(executionOrder).Updates(map[string]any{
		"state":  models.FactoryWorkOrderStateClosed,
		"result": models.FactoryWorkOrderResultArchived,
	}).Error)
	executionRun := createFinishedRun(t, support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil), models.CanvasRunResultPassed)
	linkRunExecution(t, db, r.Organization.ID, factory, line, executionOrder, executionRun.ID)

	childRun := createSubRunRecord(
		t,
		canvas.ID,
		"trigger",
		&executionRun.ID,
		&canvas.ID,
		nil,
		models.CanvasRunStateFinished,
		models.CanvasRunResultPassed,
	)

	pullRequestOrder := createTaskOrder(t, db, factory, "Pull request task")
	pullRequestRun := createFinishedRun(t, support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil), models.CanvasRunResultPassed)
	linkPullRequestRun(t, db, pullRequestOrder, pullRequestRun.ID, 11)

	sourceRun := createFinishedRun(t, support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil), models.CanvasRunResultPassed)
	sourceOrder, err := factory.CreateWorkOrder(db, "Source task", "", &r.User, nil, &sourceRun.ID)
	require.NoError(t, err)

	planningRun := createFinishedRun(t, support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil), models.CanvasRunResultPassed)
	planningOrder := createTaskOrder(t, db, factory, "Planning task")
	linkPlanningRun(t, db, r.Organization.ID, factory, canvas, planningRun, planningOrder, &r.User)

	sharedTitle := "Shared task title"
	unlinkedEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	require.NoError(t, db.Model(&unlinkedEvent).Update("custom_name", sharedTitle).Error)
	unlinkedRun := createFinishedRun(t, unlinkedEvent, models.CanvasRunResultPassed)
	_, err = factory.CreateWorkOrder(db, sharedTitle, "", &r.User, nil, nil)
	require.NoError(t, err)

	withoutTask, err := ListRuns(context.Background(), db, canvas, &pb.ListRunsRequest{})
	require.NoError(t, err)
	for _, run := range withoutTask.Runs {
		assert.Nil(t, run.Task, run.Id)
	}

	response, err := ListRuns(context.Background(), db, canvas, &pb.ListRunsRequest{IncludeTask: proto.Bool(true)})
	require.NoError(t, err)
	byID := indexRuns(response.Runs)

	assertTask(t, byID[executionRun.ID.String()], executionOrder, factory)
	assertTask(t, byID[childRun.ID.String()], executionOrder, factory)
	assertTask(t, byID[pullRequestRun.ID.String()], pullRequestOrder, factory)
	assertTask(t, byID[sourceRun.ID.String()], sourceOrder, factory)
	assertTask(t, byID[planningRun.ID.String()], planningOrder, factory)
	assert.Nil(t, byID[unlinkedRun.ID.String()].Task)

	described, err := DescribeRun(context.Background(), db, canvas, sourceRun.ID.String(), true)
	require.NoError(t, err)
	assertTask(t, described.Run, sourceOrder, factory)

	omitted, err := DescribeRun(context.Background(), db, canvas, sourceRun.ID.String(), false)
	require.NoError(t, err)
	assert.Nil(t, omitted.Run.Task)
	assert.Equal(t, sourceRun.ID.String(), omitted.Run.Id)
}

func Test__ListRuns__IncludeTaskUsesRecordedExistingTask(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "DUP")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger},
		{NodeID: "create", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "createWorkOrder"}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	canvas.FactoryID = &factory.ID
	order := createTaskOrder(t, db, factory, "Existing task")

	duplicateRoot := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	duplicate := createFinishedRun(t, duplicateRoot, models.CanvasRunResultPassed)
	duplicateExecution := createRunExecution(t, duplicate, duplicateRoot.ID, "create", models.CanvasNodeExecutionResultPassed)
	require.NoError(t, db.Model(duplicateExecution).Update("metadata", datatypes.NewJSONType(map[string]any{
		models.CanvasNodeExecutionExistingTaskIDKey: order.ID.String(),
	})).Error)

	retriedRoot := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	retried := createFinishedRun(t, retriedRoot, models.CanvasRunResultPassed)
	first := createRunExecution(t, retried, retriedRoot.ID, "create", models.CanvasNodeExecutionResultPassed)
	require.NoError(t, db.Model(first).Update("metadata", datatypes.NewJSONType(map[string]any{
		models.CanvasNodeExecutionExistingTaskIDKey: order.ID.String(),
	})).Error)
	later := createRunExecution(t, retried, retriedRoot.ID, "create", models.CanvasNodeExecutionResultPassed)
	require.NoError(t, db.Model(later).Update("created_at", time.Now().Add(time.Minute)).Error)

	response, err := ListRuns(context.Background(), db, canvas, &pb.ListRunsRequest{IncludeTask: proto.Bool(true)})
	require.NoError(t, err)
	byID := indexRuns(response.Runs)
	assertTask(t, byID[duplicate.ID.String()], order, factory)
	assert.Nil(t, byID[retried.ID.String()].Task)
}

func createTaskOrder(t *testing.T, db *gorm.DB, factory *models.Factory, title string) *models.FactoryWorkOrder {
	t.Helper()
	order, err := factory.CreateWorkOrder(db, title, "", nil, nil, nil)
	require.NoError(t, err)
	return order
}

func linkRunExecution(
	t *testing.T,
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	line *models.FactoryLine,
	order *models.FactoryWorkOrder,
	runID uuid.UUID,
) {
	t.Helper()
	dispatch := support.CreateFactoryLineDispatch(t, orgID, factory.ID, order.ID, line.ID, line.Name, nil)
	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: orgID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepName:       "implement",
		RunID:          &runID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		Result:         models.CanvasRunResultPassed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)
}

func linkPullRequestRun(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, runID uuid.UUID, number int64) {
	t.Helper()
	now := time.Now()
	pullRequest := models.FactoryPullRequest{
		ID:             uuid.New(),
		OrganizationID: order.OrganizationID,
		FactoryID:      order.FactoryID,
		WorkOrderID:    order.ID,
		Provider:       models.FactoryPullRequestProviderGitHub,
		Repository:     "acme/app",
		Number:         number,
		URL:            fmt.Sprintf("https://github.com/acme/app/pull/%d", number),
		Title:          order.Title,
		State:          models.FactoryPullRequestStateOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&pullRequest).Error)
	require.NoError(t, db.Create(&models.FactoryPullRequestRun{
		PullRequestID: pullRequest.ID,
		RunID:         runID,
		Access:        models.FactoryPullRequestAccessConcurrent,
		State:         models.FactoryPullRequestActivityStateFinished,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error)
}

func linkPlanningRun(
	t *testing.T,
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	canvas *models.Canvas,
	run *models.CanvasRun,
	order *models.FactoryWorkOrder,
	userID *uuid.UUID,
) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryPlanningSession{
		ID:               uuid.New(),
		OrganizationID:   orgID,
		FactoryID:        factory.ID,
		CreatedByUserID:  userID,
		Repository:       "acme/app",
		Kind:             models.PlanningSessionKindWorkOrderAnalysis,
		State:            models.PlanningSessionStateEnded,
		CanvasID:         &canvas.ID,
		CanvasRunID:      &run.ID,
		DraftWorkOrderID: &order.ID,
		HeartbeatAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error)
}

func indexRuns(runs []*pb.CanvasRun) map[string]*pb.CanvasRun {
	indexed := make(map[string]*pb.CanvasRun, len(runs))
	for _, run := range runs {
		indexed[run.GetId()] = run
	}
	return indexed
}

func assertTask(t *testing.T, run *pb.CanvasRun, order *models.FactoryWorkOrder, factory *models.Factory) {
	t.Helper()
	require.NotNil(t, run)
	require.NotNil(t, run.Task)
	assert.Equal(t, order.ID.String(), run.Task.Id)
	assert.Equal(t, factory.WorkOrderKey(order.Number), run.Task.Key)
	assert.Equal(t, order.Title, run.Task.Title)
}
