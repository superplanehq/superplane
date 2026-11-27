package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactoryConfigurationIDsPersistAcrossRenamesAndLineUpdates(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("workspace"), "", "")
	require.NoError(t, err)
	create := func(name string) *models.Canvas {
		now := time.Now()
		version := &models.CanvasVersion{ID: uuid.New(), Nodes: datatypes.NewJSONSlice([]models.Node{}), Edges: datatypes.NewJSONSlice([]models.Edge{}), CreatedAt: &now, UpdatedAt: &now}
		canvas := &models.Canvas{ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: &factory.ID, LiveVersionID: &version.ID, Name: name, CreatedAt: &now, UpdatedAt: &now}
		version.WorkflowID = canvas.ID
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(canvas).Error; err != nil {
				return err
			}
			return tx.Create(version).Error
		}))
		return canvas
	}
	first := create("Review Task")
	second := create("Review-Task")
	require.Equal(t, "review-task", first.ConfigurationID)
	require.Equal(t, "review-task-2", second.ConfigurationID)
	require.NotEqual(t, first.ConfigurationID, second.ConfigurationID)
	const id = "review-task"
	require.NoError(t, first.SetConfigurationID(db, id))
	require.Error(t, second.SetConfigurationID(db, id))
	name := "Renamed reviewer"
	require.NoError(t, db.Model(first).Update("name", name).Error)
	current, err := models.FindCanvasInTransaction(db, r.Organization.ID, first.ID)
	require.NoError(t, err)
	require.Equal(t, id, current.ConfigurationID)
	line, err := factory.CreateLine(db, "Delivery", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: first.ID, Entrypoint: "manual"}})
	require.NoError(t, err)
	stages := line.StagesValue()
	retained := stages[1].ConfigurationID
	require.NotEmpty(t, retained)
	stages[1].Name = "Renamed phase"
	// UI callers know the stage UUID and internal key, not its YAML identity.
	stages[1].ConfigurationID = ""
	custom := models.FactoryLineStage{ID: uuid.New(), Key: "custom-review", Name: stages[0].Name, Kind: models.FactoryLineStagePhase}
	stages = append(stages[:len(stages)-1], custom, stages[len(stages)-1])
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	require.Equal(t, retained, line.StagesValue()[1].ConfigurationID)
	ids := map[string]bool{}
	for _, stage := range line.Stages {
		require.NotEmpty(t, stage.ConfigurationID)
		require.False(t, ids[stage.ConfigurationID])
		ids[stage.ConfigurationID] = true
	}
	// Raw creation also stores stages once, even when another line shares its runner.
	other := &models.FactoryLine{
		ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID,
		Name: "Alternate delivery", Steps: line.Steps,
	}
	require.NoError(t, db.Create(other).Error)
	require.Len(t, other.Stages, 4)
	for _, stage := range other.Stages {
		require.NotEmpty(t, stage.ConfigurationID)
		require.False(t, ids[stage.ConfigurationID], "stage IDs are unique across workspace lines")
	}
	current, err = models.FindCanvasInTransaction(db, r.Organization.ID, first.ID)
	require.NoError(t, err)
	require.Equal(t, id, current.ConfigurationID)

}
