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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__FactoryLine__StartStep__CreatesPendingRunAndExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Fix login bug", "", uuid.New(), nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(database.Conn(), "bug", nil)
	require.NoError(t, err)

	now := time.Now()
	liveVersionID := uuid.New()
	factoryID := factory.ID
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           support.RandomName("factory-app"),
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	entrypoint := "start-work"
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}

		node := models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     entrypoint,
			Name:       "Start work",
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onRun"},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}

		version := models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{
				{
					ID:   entrypoint,
					Name: "Start work",
					Type: models.NodeTypeTrigger,
					Ref: models.NodeRef{
						Trigger: &models.TriggerRef{Name: "onRun"},
					},
				},
			}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))

	steps := []models.FactoryLineStep{{
		Name:       "start-work",
		Type:       models.FactoryLineStepTypeRunApp,
		AppID:      canvas.ID,
		Entrypoint: entrypoint,
	}}
	require.NoError(t, line.Update(database.Conn(), nil, steps))

	var result *models.FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var startErr error
		result, startErr = line.StartStep(tx, order, 0)
		return startErr
	}))

	require.NotNil(t, result)
	assert.Equal(t, models.CanvasRunStatePending, result.Run.State)
	assert.Equal(t, canvas.ID, result.Run.WorkflowID)
	assert.Equal(t, entrypoint, result.Run.NodeID)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, result.Execution.Status)
	assert.Equal(t, 0, result.Execution.StepIndex)
	assert.Equal(t, "start-work", result.Execution.StepName)
}
