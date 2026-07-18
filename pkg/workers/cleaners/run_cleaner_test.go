package cleaners_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/cleaners"
	"gorm.io/gorm"
)

func Test__RunCleanerOptions_Validate(t *testing.T) {
	t.Run("retention mode requires reference time", func(t *testing.T) {
		_, err := cleaners.NewRunCleaner(nil, cleaners.RunCleanerOptions{
			Mode: cleaners.RunCleanerModeRetention,
		})
		require.Error(t, err)
	})

	t.Run("canvas teardown mode requires workflow id", func(t *testing.T) {
		_, err := cleaners.NewRunCleaner(nil, cleaners.RunCleanerOptions{
			Mode: cleaners.RunCleanerModeCanvasTeardown,
		})
		require.Error(t, err)
	})
}

func Test__RunCleaner_CleanBatch_CanvasTeardown(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, err := models.CreateOrganization("org-"+uuid.NewString(), "test org")
	require.NoError(t, err)

	now := time.Now()
	versionID := uuid.New()
	canvas := models.Canvas{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		LiveVersionID:  &versionID,
		Name:           "canvas-" + uuid.NewString(),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}

		version := models.CanvasVersion{
			ID:         versionID,
			WorkflowID: canvas.ID,
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}

		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		for _, nodeID := range []string{"trigger", "component"} {
			node := models.CanvasNode{
				WorkflowID: canvas.ID,
				NodeID:     nodeID,
				Type:       models.NodeTypeComponent,
				CreatedAt:  &now,
				UpdatedAt:  &now,
			}
			if err := tx.Create(&node).Error; err != nil {
				return err
			}
		}

		return nil
	}))

	rootEvent := models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     "trigger",
		Channel:    "default",
		Data:       models.NewJSONValue(map[string]any{}),
		State:      models.CanvasEventStateRouted,
		CreatedAt:  &now,
	}
	require.NoError(t, database.Conn().Create(&rootEvent).Error)

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), rootEvent.ID)
	require.NoError(t, err)

	execution := models.CanvasNodeExecution{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      "component",
		RootEventID: rootEvent.ID,
		RunID:       run.ID,
		EventID:     rootEvent.ID,
		State:       models.CanvasNodeExecutionStateFinished,
		Result:      models.CanvasNodeExecutionResultPassed,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)

	var deleted int
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		runCleaner, err := cleaners.NewRunCleaner(tx, cleaners.RunCleanerOptions{
			Mode:       cleaners.RunCleanerModeCanvasTeardown,
			WorkflowID: canvas.ID,
		})
		if err != nil {
			return err
		}

		deleted, err = runCleaner.CleanBatch(10)
		return err
	}))

	require.Equal(t, 1, deleted)

	remainingRuns, err := canvas.CountRuns(database.Conn())
	require.NoError(t, err)
	require.Equal(t, int64(0), remainingRuns)
}
