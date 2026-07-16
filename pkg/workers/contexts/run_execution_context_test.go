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
			{NodeID: "onInvoke1", Name: "On Invoke 1", Type: models.NodeTypeTrigger},
			{NodeID: "onInvoke2", Name: "On Invoke 2", Type: models.NodeTypeTrigger},
			{
				NodeID: "invokeApp",
				Name:   "Invoke App",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "invokeApp"}}),
			},
		},
		nil,
	)

	rootRun := createRunRecord(t, canvas.ID, "onInvoke1", nil)
	runOnInvoke2 := createRunRecord(t, canvas.ID, "onInvoke2", &rootRun.ID)
	parentRun := createRunRecord(t, canvas.ID, "onInvoke1", &runOnInvoke2.ID)
	execution := createRunExecution(t, canvas.ID, parentRun.ID, nodes[2].NodeID)

	ctx := NewRunExecutionContext(database.Conn(), canvas, &nodes[2], execution)
	_, err := ctx.Create(core.RunCreationParams{
		App:  canvas.ID.String(),
		Node: "onInvoke2",
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
			{NodeID: "onInvoke", Name: "On Invoke", Type: models.NodeTypeTrigger},
		},
		nil,
	)

	parentRun := createRunRecord(t, canvas.ID, "forEach", nil)
	createRunRecord(t, canvas.ID, "onInvoke", &parentRun.ID)
	execution := createRunExecution(t, canvas.ID, parentRun.ID, nodes[0].NodeID)

	ctx := NewRunExecutionContext(database.Conn(), canvas, &nodes[0], execution)
	run, err := ctx.Create(core.RunCreationParams{
		App:  canvas.ID.String(),
		Node: "onInvoke",
	})

	require.NoError(t, err)
	require.NotNil(t, run)
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

func createRunExecution(t *testing.T, workflowID, runID uuid.UUID, nodeID string) *models.CanvasNodeExecution {
	t.Helper()

	rootEvent := support.EmitCanvasEventForNode(t, workflowID, nodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, workflowID, nodeID, rootEvent.ID, rootEvent.ID)
	execution.RunID = runID
	require.NoError(t, database.Conn().Save(execution).Error)
	return execution
}
