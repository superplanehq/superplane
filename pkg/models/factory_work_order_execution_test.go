package models_test

import (
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

func Test__ListFactoryWorkOrderExecutionsByLineDispatchIDs__IncludesExecutionsAfterRunDeleted(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "implement",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		Result:         models.CanvasRunResultPassed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&execution).Error)

	listed, err := models.ListFactoryWorkOrderExecutionsByLineDispatchIDs(db, []uuid.UUID{dispatch.ID})
	require.NoError(t, err)
	require.Len(t, listed[dispatch.ID], 1)
	require.NotNil(t, listed[dispatch.ID][0].RunID)
	assert.Equal(t, run.ID, *listed[dispatch.ID][0].RunID)
	assert.Equal(t, "implement", listed[dispatch.ID][0].StepName)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := run.DeleteChain(tx)
		return err
	}))

	listed, err = models.ListFactoryWorkOrderExecutionsByLineDispatchIDs(db, []uuid.UUID{dispatch.ID})
	require.NoError(t, err)
	require.Len(t, listed[dispatch.ID], 1)
	assert.Nil(t, listed[dispatch.ID][0].RunID)
	assert.Nil(t, listed[dispatch.ID][0].CanvasID)
	assert.Equal(t, "implement", listed[dispatch.ID][0].StepName)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, listed[dispatch.ID][0].Status)
	assert.Equal(t, models.CanvasRunResultPassed, listed[dispatch.ID][0].Result)
}
