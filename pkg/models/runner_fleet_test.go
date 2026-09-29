package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestCreateDefaultInstallationRunnerFleetsIsIdempotent(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	require.NoError(t, CreateDefaultInstallationRunnerFleets(
		database.Conn(),
		"v1.0.0",
	))
	require.NoError(t, CreateDefaultInstallationRunnerFleets(
		database.Conn(),
		"v2.0.0",
	))

	var fleets []RunnerFleet
	require.NoError(t, database.Conn().
		Where("scope_type = ?", RunnerFleetScopeInstallation).
		Order("slug").
		Find(&fleets).Error)
	require.Len(t, fleets, 4)
	for _, fleet := range fleets {
		require.Equal(t, "v1.0.0", fleet.RunnerVersion)
	}
}
