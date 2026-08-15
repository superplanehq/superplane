package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TestFactoryLine_StartStep_SnapshotsLineSteps verifies that a
// FactoryWorkOrderExecution's LineSteps is captured once, at StartStep time,
// and stays unchanged after the containing line is edited -- this is the
// core guarantee behind "snapshot factory line steps into the work-order
// context": history rendering must not be rewritten by later line edits.
func TestFactoryLine_StartStep_SnapshotsLineSteps(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, userID, factoryModel := setupFactoryWithUser(t, "snapshot-line-steps")

	firstApp, firstEntry := createLineStepApp(t, organization.ID, userID, "step-one", "start-one")
	secondApp, secondEntry := createLineStepApp(t, organization.ID, userID, "step-two", "start-two")

	originalSteps := []FactoryLineStep{
		{Name: "intake", Type: FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Name: "implement", Type: FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	}

	line, err := factoryModel.CreateLine(database.Conn(), "ship", originalSteps)
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Ship feature", "", &userID, nil, nil)
	require.NoError(t, err)

	var result *FactoryLineStepResult
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var startErr error
		result, startErr = line.StartStep(tx, order, 0)
		return startErr
	}))
	require.NotNil(t, result)

	snapshot := []FactoryLineStep(result.Execution.LineSteps)
	require.Len(t, snapshot, 2)
	assert.Equal(t, "intake", snapshot[0].Name)
	assert.Equal(t, "implement", snapshot[1].Name)

	// Edit the line after the step started: rename steps and drop one.
	editedSteps := []FactoryLineStep{
		{Name: "renamed-only-step", Type: FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
	}
	require.NoError(t, line.Update(database.Conn(), nil, editedSteps))

	reloaded, err := FindWorkOrderExecutionByRunID(database.Conn(), result.Run.ID)
	require.NoError(t, err)
	reloadedSnapshot := []FactoryLineStep(reloaded.LineSteps)
	require.Len(t, reloadedSnapshot, 2, "execution's snapshot must not shrink when the line is edited")
	assert.Equal(t, "intake", reloadedSnapshot[0].Name)
	assert.Equal(t, "implement", reloadedSnapshot[1].Name)

	byWorkOrder, err := ListFactoryWorkOrderExecutionsByWorkOrderIDs(database.Conn(), []uuid.UUID{order.ID})
	require.NoError(t, err)
	records := byWorkOrder[order.ID]
	require.Len(t, records, 1)

	listedSnapshot := []FactoryLineStep(records[0].LineSteps)
	require.Len(t, listedSnapshot, 2, "listed execution's snapshot must not reflect the edited line")
	assert.Equal(t, "intake", listedSnapshot[0].Name)
	assert.Equal(t, "implement", listedSnapshot[1].Name)
}

func createLineStepApp(
	t *testing.T,
	organizationID uuid.UUID,
	userID uuid.UUID,
	name, entrypoint string,
) (*Canvas, string) {
	t.Helper()

	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &Canvas{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		LiveVersionID:  &liveVersionID,
		Name:           fmt.Sprintf("%s-%d", name, time.Now().UnixNano()),
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}

		node := CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     entrypoint,
			Name:       name,
			Type:       NodeTypeTrigger,
			State:      CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(NodeRef{
				Trigger: &TriggerRef{Name: onRunTriggerName},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}

		version := CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &userID,
			Nodes: datatypes.NewJSONSlice([]Node{
				{
					ID:   entrypoint,
					Name: name,
					Type: NodeTypeTrigger,
					Ref: NodeRef{
						Trigger: &TriggerRef{Name: onRunTriggerName},
					},
				},
			}),
			Edges:     datatypes.NewJSONSlice([]Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))

	return canvas, entrypoint
}
