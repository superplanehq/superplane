package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryLine__ColumnColors__DefaultsToScreenshotColors(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	expected := map[string]string{"phase-0": "sky", "verify": "yellow", "done": "lime"}
	assert.Equal(t, expected, line.ColumnColorsValue())

	// Round-trip through storage so we catch a NULL insert into the
	// NOT NULL column_colors column (the DB default is not applied when
	// GORM sends an explicit null for an unset JSONType field).
	reloaded, err := factory.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, expected, reloaded.ColumnColorsValue())
}

func Test__FactoryLine__Stages__DefaultAndRoundTrip(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry,
	}})
	require.NoError(t, err)

	require.Len(t, line.StagesValue(), 4)
	assert.Equal(t, []string{"backlog", "phase", "verify", "done"}, []string{
		line.StagesValue()[0].Kind,
		line.StagesValue()[1].Kind,
		line.StagesValue()[2].Kind,
		line.StagesValue()[3].Kind,
	})
	assert.Contains(t, line.StagesValue()[1].Name, "Implement")
	assert.Equal(t, 0, *line.StagesValue()[1].StepIndex)

	reloaded, err := factory.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, line.StagesValue(), reloaded.StagesValue())
}

func Test__FactoryLine__Update__ReconcilesPhasesAndKeepsCustomStages(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	first, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start-one")
	second, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Release", "start-two")
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: first.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: second.ID, Entrypoint: secondEntry},
	}
	line, err := factory.CreateLine(db, "ship", steps)
	require.NoError(t, err)
	custom := models.FactoryLineStage{ID: uuid.New(), Key: "custom-qa", Name: "QA", Kind: models.FactoryLineStageCustom}
	stages := append([]models.FactoryLineStage{}, line.StagesValue()[:2]...)
	stages = append(stages, custom)
	stages = append(stages, line.StagesValue()[2:]...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	require.NoError(t, line.Update(db, nil, nil, map[string]string{"phase-1": "purple", "custom-qa": "teal"}))

	require.NoError(t, line.Update(db, nil, steps[:1], nil))

	assert.Equal(t, []string{"backlog", "phase-0", "custom-qa", "verify", "done"}, []string{
		line.StagesValue()[0].Key,
		line.StagesValue()[1].Key,
		line.StagesValue()[2].Key,
		line.StagesValue()[3].Key,
		line.StagesValue()[4].Key,
	})
	assert.Equal(t, custom.ID, line.StagesValue()[2].ID)
	assert.Equal(t, map[string]string{"custom-qa": "teal"}, line.ColumnColorsValue())
}

func Test__FactoryLine__Update__KeepsOptionalStagesRemoved(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
	steps := []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}}
	line, err := factory.CreateLine(db, "ship", steps)
	require.NoError(t, err)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, []models.FactoryLineStage{
		line.StagesValue()[0],
		line.StagesValue()[len(line.StagesValue())-1],
	}))

	require.NoError(t, line.Update(db, nil, steps, nil))

	assert.Equal(t, []string{"backlog", "done"}, []string{line.StagesValue()[0].Key, line.StagesValue()[1].Key})
}

func Test__FactoryLine__Update__PersistsColumnColors(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	colors := map[string]string{"backlog": "lime", "verify": "teal"}
	require.NoError(t, line.Update(db, nil, nil, colors))
	assert.Equal(t, colors, line.ColumnColorsValue())

	// Reload from the database to make sure the value round-trips through
	// storage, not just the in-memory struct.
	reloaded, err := factory.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, colors, reloaded.ColumnColorsValue())
}

func Test__FactoryLine__Update__ColumnColorsOnlyDoesNotDisturbNameOrSteps(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}
	line, err := factory.CreateLine(db, "ship", steps)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, map[string]string{"backlog": "lime"}))

	assert.Equal(t, "ship", line.Name)
	require.Len(t, line.Steps, 1)
	assert.Equal(t, entry, line.Steps[0].Entrypoint)
	assert.Equal(t, map[string]string{"backlog": "lime"}, line.ColumnColorsValue())
}

func Test__FactoryLine__Update__NilColumnColorsLeavesStoredValueUnchanged(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, map[string]string{"backlog": "lime"}))

	newName := "shipped"
	require.NoError(t, line.Update(db, &newName, nil, nil))

	assert.Equal(t, newName, line.Name)
	assert.Equal(t, map[string]string{"backlog": "lime"}, line.ColumnColorsValue())
}

func Test__FactoryLine__Update__EmptyColumnColorsClearsStoredValue(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, map[string]string{"backlog": "lime"}))
	require.NoError(t, line.Update(db, nil, nil, map[string]string{}))

	assert.Equal(t, map[string]string{}, line.ColumnColorsValue())
}
