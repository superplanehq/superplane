package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCreateFactory_DefaultsPlanningOnWithoutSetup(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	assert.Equal(t, models.DefaultFactoryPlanning(), factory.Planning())

	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, models.DefaultFactoryPlanning(), reloaded.Planning())
}

func TestFactory_UpdatePlanningKeepsScoreFlagsWhenOff(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	require.NoError(t, factory.UpdatePlanning(db, models.FactoryPlanning{
		Enabled:        false,
		Clarity:        true,
		Confidence:     false,
		SetupCompleted: true,
	}))
	assert.Equal(t, models.FactoryPlanning{
		Enabled:        false,
		Clarity:        true,
		Confidence:     false,
		SetupCompleted: true,
	}, factory.Planning())

	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPlanning{
		Enabled:        false,
		Clarity:        true,
		Confidence:     false,
		SetupCompleted: true,
	}, reloaded.Planning())
}

func TestFactory_UpdatePlanningPersistsAutoStartLine(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	require.NoError(t, factory.UpdatePlanning(db, models.FactoryPlanning{
		Enabled:        true,
		Confidence:     true,
		SetupCompleted: true,
		AutoStart:      true,
		AutoStartLine:  "  ship  ",
	}))
	assert.Equal(t, models.FactoryPlanning{
		Enabled:        true,
		Confidence:     true,
		SetupCompleted: true,
		AutoStart:      true,
		AutoStartLine:  line.Name,
	}, factory.Planning())

	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, factory.Planning(), reloaded.Planning())
}

func TestFactory_UpdatePlanningClearsAutoStartWhenConfidenceIsOff(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	require.NoError(t, factory.UpdatePlanning(db, models.FactoryPlanning{
		Enabled:       true,
		Confidence:    false,
		AutoStart:     true,
		AutoStartLine: "missing",
	}))
	assert.False(t, factory.Planning().AutoStart)
	assert.Equal(t, "missing", factory.Planning().AutoStartLine)
}

func TestFactory_UpdatePlanningRejectsAutoStartWithoutKnownLine(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	err = factory.UpdatePlanning(db, models.FactoryPlanning{
		Enabled:    true,
		Confidence: true,
		AutoStart:  true,
	})
	assert.ErrorIs(t, err, models.ErrFactoryPlanningAutoStartLineRequired)

	err = factory.UpdatePlanning(db, models.FactoryPlanning{
		Enabled:       true,
		Confidence:    true,
		AutoStart:     true,
		AutoStartLine: "missing",
	})
	assert.ErrorIs(t, err, models.ErrFactoryPlanningAutoStartLineUnknown)
}
