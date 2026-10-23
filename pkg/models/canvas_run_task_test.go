package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestResolveCanvasRunTasks_UsesNearestAncestorInLinkOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "TASK")
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
	canvas.FactoryID = &factory.ID

	child, parent := createParentAndChildRuns(t, canvas)
	childOrder := createTaskOrder(t, db, factory, "Child execution")
	parentOrder := createTaskOrder(t, db, factory, "Parent execution")
	linkRunExecution(t, db, r.Organization.ID, factory, line, childOrder, child.ID)
	linkRunExecution(t, db, r.Organization.ID, factory, line, parentOrder, parent.ID)

	labels, err := models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{child})
	require.NoError(t, err)
	assert.Equal(t, childOrder.ID, labels[child.ID].ID)

	pullRequestChild, executionParent := createParentAndChildRuns(t, canvas)
	pullRequestOrder := createTaskOrder(t, db, factory, "Near pull request")
	executionOrder := createTaskOrder(t, db, factory, "Far execution")
	linkPullRequestRun(t, db, pullRequestOrder, pullRequestChild.ID, 21)
	linkRunExecution(t, db, r.Organization.ID, factory, line, executionOrder, executionParent.ID)

	labels, err = models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{pullRequestChild})
	require.NoError(t, err)
	assert.Equal(t, executionOrder.ID, labels[pullRequestChild.ID].ID)
	assert.Equal(t, factory.WorkOrderKey(executionOrder.Number), labels[pullRequestChild.ID].Key)

	sourceRun := createStoredRun(t, canvas, nil)
	sourceOrder, err := factory.CreateWorkOrder(db, "Source task", "", &r.User, nil, &sourceRun.ID)
	require.NoError(t, err)
	planningOrder := createTaskOrder(t, db, factory, "Planning task")
	linkPlanningRun(t, db, r, factory, canvas, sourceRun, planningOrder)

	withoutFactory := *canvas
	withoutFactory.FactoryID = nil
	labels, err = models.ResolveCanvasRunTasks(db, &withoutFactory, []models.CanvasRun{sourceRun})
	require.NoError(t, err)
	assert.Equal(t, planningOrder.ID, labels[sourceRun.ID].ID)

	labels, err = models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{sourceRun})
	require.NoError(t, err)
	assert.Equal(t, sourceOrder.ID, labels[sourceRun.ID].ID)
}

func TestResolveCanvasRunTasks_StopsOnCycleAndAncestorBound(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "CYCL")
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
	canvas.FactoryID = &factory.ID

	first := createStoredRun(t, canvas, nil)
	second := createStoredRun(t, canvas, &first.ID)
	require.NoError(t, db.Model(&first).Update("parent_run_id", second.ID).Error)
	first.ParentRunID = &second.ID
	order := createTaskOrder(t, db, factory, "Cycle task")
	linkRunExecution(t, db, r.Organization.ID, factory, line, order, second.ID)

	labels, err := models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{first})
	require.NoError(t, err)
	assert.Equal(t, order.ID, labels[first.ID].ID)

	chain := createRunChain(t, canvas, 33)
	insideBound := createTaskOrder(t, db, factory, "Inside bound")
	outsideBound := createTaskOrder(t, db, factory, "Outside bound")
	linkRunExecution(t, db, r.Organization.ID, factory, line, insideBound, chain[31].ID)
	linkRunExecution(t, db, r.Organization.ID, factory, line, outsideBound, chain[32].ID)

	labels, err = models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{chain[0]})
	require.NoError(t, err)
	assert.Equal(t, insideBound.ID, labels[chain[0].ID].ID)

	onlyOutside := createRunChain(t, canvas, 33)
	missed := createTaskOrder(t, db, factory, "Too far")
	linkRunExecution(t, db, r.Organization.ID, factory, line, missed, onlyOutside[32].ID)
	labels, err = models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{onlyOutside[0]})
	require.NoError(t, err)
	_, found := labels[onlyOutside[0].ID]
	assert.False(t, found)
}

func createParentAndChildRuns(t *testing.T, canvas *models.Canvas) (models.CanvasRun, models.CanvasRun) {
	t.Helper()
	parent := createStoredRun(t, canvas, nil)
	child := createStoredRun(t, canvas, &parent.ID)
	return child, parent
}

func createRunChain(t *testing.T, canvas *models.Canvas, length int) []models.CanvasRun {
	t.Helper()
	runs := make([]models.CanvasRun, length)
	var parent *uuid.UUID
	for i := length - 1; i >= 0; i-- {
		runs[i] = createStoredRun(t, canvas, parent)
		id := runs[i].ID
		parent = &id
	}
	return runs
}

func createStoredRun(t *testing.T, canvas *models.Canvas, parent *uuid.UUID) models.CanvasRun {
	t.Helper()
	now := time.Now()
	run := models.CanvasRun{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      "trigger",
		VersionID:   *canvas.LiveVersionID,
		ParentRunID: parent,
		State:       models.CanvasRunStateFinished,
		Result:      models.CanvasRunResultPassed,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	require.NoError(t, database.DB(t.Context()).Create(&run).Error)
	return run
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
	r *support.ResourceRegistry,
	factory *models.Factory,
	canvas *models.Canvas,
	run models.CanvasRun,
	order *models.FactoryWorkOrder,
) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryPlanningSession{
		ID:               uuid.New(),
		OrganizationID:   r.Organization.ID,
		FactoryID:        factory.ID,
		CreatedByUserID:  &r.User,
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
