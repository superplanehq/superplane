package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRunCancellationNotifier_BindCollectsCancelledChildRuns(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Name:   "Run App",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)

	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createCancellationTestRun(t, parentCanvas.ID, "trigger", nil)
	execution := createCancellationTestExecution(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)
	require.NoError(t, execution.RequestCancellation(database.Conn(), &r.User))

	execution, err := models.FindNodeExecution(parentCanvas.ID, execution.ID)
	require.NoError(t, err)

	childRun := createCancellationTestChildRun(t, childCanvas.ID, "onRun", parentRun.ID, parentCanvas.ID, execution.ID, models.CanvasRunStatePending)

	notifier := &RunCancellationNotifier{}
	ctx := notifier.Bind(contexts.NewRunExecutionContext(database.Conn(), parentCanvas, &parentNodes[1], execution))
	require.NoError(t, ctx.Cancel())

	require.Len(t, notifier.Outcomes, 1)
	assert.Equal(t, childCanvas.ID, notifier.Outcomes[0].WorkflowID)
	assert.Equal(t, childRun.ID, notifier.Outcomes[0].RunID)
	require.NotNil(t, notifier.Outcomes[0].DrainResult)
}

func TestRunCancellationNotifier_PublishNilIsNoOp(t *testing.T) {
	var notifier *RunCancellationNotifier
	require.NotPanics(t, notifier.Publish)
}

func TestRunCancellationNotifier_BindNilIsNoOp(t *testing.T) {
	ctx := (*RunCancellationNotifier)(nil).Bind(contexts.NewRunExecutionContext(nil, nil, nil, nil))
	require.NotNil(t, ctx)
}

func createCancellationTestRun(t *testing.T, workflowID uuid.UUID, nodeID string, parentRunID *uuid.UUID) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		VersionID:   liveVersion.ID,
		ParentRunID: parentRunID,
		State:       models.CanvasRunStateStarted,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createCancellationTestChildRun(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	parentRunID uuid.UUID,
	parentWorkflowID uuid.UUID,
	parentExecutionID uuid.UUID,
	state string,
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
		ParentRunID:       &parentRunID,
		ParentWorkflowID:  &parentWorkflowID,
		ParentExecutionID: &parentExecutionID,
		State:             state,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createCancellationTestExecution(t *testing.T, workflowID, runID uuid.UUID, nodeID string) *models.CanvasNodeExecution {
	t.Helper()

	rootEvent := support.EmitCanvasEventForNode(t, workflowID, nodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, workflowID, nodeID, rootEvent.ID, rootEvent.ID)
	execution.RunID = runID
	require.NoError(t, database.Conn().Save(execution).Error)
	return execution
}
