package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func Test__Integration_MarkError(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)

	integration, err := CreateIntegration(uuid.New(), organization.ID, "dummy", "integration-"+uuid.NewString(), nil)
	require.NoError(t, err)
	integration.State = IntegrationStateReady
	require.NoError(t, database.Conn().Save(integration).Error)

	//
	// Simulate a concurrent write to a column MarkError does not own (e.g. a
	// rename via the update-integration endpoint) happening after this
	// in-memory copy was loaded, but before MarkError persists.
	//
	concurrentName := "renamed-" + uuid.NewString()
	require.NoError(t, database.Conn().Model(&Integration{}).
		Where("id = ?", integration.ID).
		Update("installation_name", concurrentName).Error)

	require.NoError(t, integration.MarkError(database.Conn(), "credentials are invalid or expired"))

	assert.Equal(t, IntegrationStateError, integration.State)
	assert.Equal(t, "credentials are invalid or expired", integration.StateDescription)

	reloaded, err := FindUnscopedIntegration(integration.ID)
	require.NoError(t, err)
	assert.Equal(t, IntegrationStateError, reloaded.State)
	assert.Equal(t, "credentials are invalid or expired", reloaded.StateDescription)

	// The concurrent rename must survive MarkError's narrowed update.
	assert.Equal(t, concurrentName, reloaded.InstallationName)
}
