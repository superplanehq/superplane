package contexts

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__RunExecutionContext__Create__RejectsEntrypointCycle(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "run1", Name: "Run 1", Type: models.NodeTypeTrigger},
			{NodeID: "run2", Name: "Run 2", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Name:   "Run App",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)

	rootRun := createRunRecord(t, canvas.ID, "run1", nil)
	run2 := createRunRecord(t, canvas.ID, "run2", &rootRun.ID)
	parentRun := createRunRecord(t, canvas.ID, "run1", &run2.ID)
	execution := createRunExecution(t, canvas.ID, parentRun.ID, nodes[2].NodeID)

	ctx := NewRunExecutionContext(database.Conn(), canvas, &nodes[2], execution)
	_, err := ctx.Create(core.RunCreationParams{
		App:  canvas.ID.String(),
		Node: "run2",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrSubRunEntrypointCycle))
}

func Test__RunExecutionContext__Create__AllowsSiblingSubRuns(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "forEach", Name: "For Each", Type: models.NodeTypeComponent},
			{NodeID: "onRun", Name: "On Run", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	parentRun := createRunRecord(t, canvas.ID, "forEach", nil)
	createRunRecord(t, canvas.ID, "onRun", &parentRun.ID)
	execution := createRunExecution(t, canvas.ID, parentRun.ID, nodes[0].NodeID)

	ctx := NewRunExecutionContext(database.Conn(), canvas, &nodes[0], execution)
	run, err := ctx.Create(core.RunCreationParams{
		App:  canvas.ID.String(),
		Node: "onRun",
	})

	require.NoError(t, err)
	require.NotNil(t, run)
}

func Test__RunExecutionContext__Cancel__CancelsChildRuns(t *testing.T) {
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

	parentRun := createRunRecord(t, parentCanvas.ID, "trigger", nil)
	execution := createRunExecution(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)
	require.NoError(t, execution.RequestCancellation(database.Conn(), &r.User))

	execution, err := models.FindNodeExecution(parentCanvas.ID, execution.ID)
	require.NoError(t, err)

	childRun := createChildRunRecord(t, childCanvas.ID, "onRun", parentRun.ID, parentCanvas.ID, execution.ID, models.CanvasRunStatePending)

	ctx := NewRunExecutionContext(database.Conn(), parentCanvas, &parentNodes[1], execution)
	require.NoError(t, ctx.Cancel())

	updatedChild, err := models.FindCanvasRunInTransaction(database.Conn(), childCanvas.ID, childRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedChild.State)
}

func createRunRecord(t *testing.T, workflowID uuid.UUID, nodeID string, parentRunID *uuid.UUID) *models.CanvasRun {
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

func createChildRunRecord(
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

func createRunExecution(t *testing.T, workflowID, runID uuid.UUID, nodeID string) *models.CanvasNodeExecution {
	t.Helper()

	rootEvent := support.EmitCanvasEventForNode(t, workflowID, nodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, workflowID, nodeID, rootEvent.ID, rootEvent.ID)
	execution.RunID = runID
	require.NoError(t, database.Conn().Save(execution).Error)
	return execution
}
