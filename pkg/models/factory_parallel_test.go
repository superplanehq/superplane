package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
)

func TestResolveOrganizationFactoryMaxParallelTasks(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	max, err := ResolveOrganizationFactoryMaxParallelTasks(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, DefaultFactoryMaxParallelTasks, max)

	require.NoError(t, SetInstallationMaxParallelFactoryTasks(database.Conn(), 20))
	max, err = ResolveOrganizationFactoryMaxParallelTasks(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, 20, max)

	override := 3
	require.NoError(t, SetOrganizationMaxParallelFactoryTasks(database.Conn(), r.Organization.ID, &override))
	max, err = ResolveOrganizationFactoryMaxParallelTasks(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, max)

	require.NoError(t, SetOrganizationMaxParallelFactoryTasks(database.Conn(), r.Organization.ID, nil))
	max, err = ResolveOrganizationFactoryMaxParallelTasks(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, 20, max)
}

func TestSetMaxParallelFactoryTasksRejectsZero(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	err := SetInstallationMaxParallelFactoryTasks(database.Conn(), 0)
	require.ErrorIs(t, err, ErrMaxParallelFactoryTasksInvalid)

	zero := 0
	err = SetOrganizationMaxParallelFactoryTasks(database.Conn(), r.Organization.ID, &zero)
	require.ErrorIs(t, err, ErrMaxParallelFactoryTasksInvalid)
}
