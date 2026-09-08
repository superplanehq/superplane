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
	require.NoError(t, line.Update(db, nil, nil, colors, nil))
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

	require.NoError(t, line.Update(db, nil, nil, map[string]string{"backlog": "lime"}, nil))

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

	require.NoError(t, line.Update(db, nil, nil, map[string]string{"backlog": "lime"}, nil))

	newName := "shipped"
	require.NoError(t, line.Update(db, &newName, nil, nil, nil))

	assert.Equal(t, newName, line.Name)
	assert.Equal(t, map[string]string{"backlog": "lime"}, line.ColumnColorsValue())
}

func Test__FactoryLine__ColumnAutomations__DefaultsToEmptyMap(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	assert.Equal(t, map[string][]uuid.UUID{}, line.ColumnAutomationsValue())

	reloaded, err := factory.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, map[string][]uuid.UUID{}, reloaded.ColumnAutomationsValue())
}

func Test__FactoryLine__Update__PersistsColumnAutomations(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	bindings := map[string][]uuid.UUID{"phase-0": {app.ID}}
	require.NoError(t, line.Update(db, nil, nil, nil, bindings))
	assert.Equal(t, bindings, line.ColumnAutomationsValue())

	reloaded, err := factory.FindLine(db, line.ID)
	require.NoError(t, err)
	assert.Equal(t, bindings, reloaded.ColumnAutomationsValue())
}

func Test__FactoryLine__Update__ColumnAutomationsOnlyDoesNotDisturbNameOrSteps(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
	extra, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}
	line, err := factory.CreateLine(db, "ship", steps)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, nil, map[string][]uuid.UUID{"phase-0": {extra.ID}}))

	assert.Equal(t, "ship", line.Name)
	require.Len(t, line.Steps, 1)
	assert.Equal(t, entry, line.Steps[0].Entrypoint)
	assert.Equal(t, map[string][]uuid.UUID{"phase-0": {extra.ID}}, line.ColumnAutomationsValue())
}

func Test__FactoryLine__Update__NilColumnAutomationsLeavesStoredValueUnchanged(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, nil, map[string][]uuid.UUID{"backlog": {app.ID}}))

	newName := "shipped"
	require.NoError(t, line.Update(db, &newName, nil, nil, nil))

	assert.Equal(t, newName, line.Name)
	assert.Equal(t, map[string][]uuid.UUID{"backlog": {app.ID}}, line.ColumnAutomationsValue())
}

func Test__FactoryLine__Update__EmptyColumnAutomationsClearsStoredValue(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, nil, map[string][]uuid.UUID{"backlog": {app.ID}}))
	require.NoError(t, line.Update(db, nil, nil, nil, map[string][]uuid.UUID{}))

	assert.Equal(t, map[string][]uuid.UUID{}, line.ColumnAutomationsValue())
}

func Test__FactoryLine__Update__EmptyColumnColorsClearsStoredValue(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	require.NoError(t, line.Update(db, nil, nil, map[string]string{"backlog": "lime"}, nil))
	require.NoError(t, line.Update(db, nil, nil, map[string]string{}, nil))

	assert.Equal(t, map[string]string{}, line.ColumnColorsValue())
}
