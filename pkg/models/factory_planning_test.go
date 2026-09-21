package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCreateFactory_DefaultsPlanningOn(t *testing.T) {
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
		Enabled:    false,
		Clarity:    true,
		Confidence: false,
	}))
	assert.Equal(t, models.FactoryPlanning{Enabled: false, Clarity: true, Confidence: false}, factory.Planning())

	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPlanning{Enabled: false, Clarity: true, Confidence: false}, reloaded.Planning())
}
