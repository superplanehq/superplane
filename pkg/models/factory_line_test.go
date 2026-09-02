package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryLine__ColumnColors__DefaultsToEmptyMap(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{}, line.ColumnColorsValue())

	// Round-trip through storage so we catch a NULL insert into the
	// NOT NULL column_colors column (the DB default is not applied when
	// GORM sends an explicit null for an unset JSONType field).
	reloaded, err := factory.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{}, reloaded.ColumnColorsValue())
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
