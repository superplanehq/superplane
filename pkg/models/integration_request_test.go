package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func Test__ResetStuckProcessingIntegrationRequests(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)

	integration, err := CreateIntegration(uuid.New(), organization.ID, "dummy", "integration-"+uuid.NewString(), nil)
	require.NoError(t, err)

	now := time.Now()

	//
	// A request stuck in "processing" longer than the timeout should be reset.
	//
	stale := &IntegrationRequest{
		ID:                uuid.New(),
		AppInstallationID: integration.ID,
		State:             IntegrationRequestStateProcessing,
		Type:              IntegrationRequestTypeSync,
		RunAt:             now,
		CreatedAt:         now.Add(-time.Hour),
		UpdatedAt:         now.Add(-20 * time.Minute),
	}
	require.NoError(t, database.Conn().Create(stale).Error)

	//
	// A request that just started processing must be left alone (it may still be
	// in flight on another replica during a rolling deploy).
	//
	fresh := &IntegrationRequest{
		ID:                uuid.New(),
		AppInstallationID: integration.ID,
		State:             IntegrationRequestStateProcessing,
		Type:              IntegrationRequestTypeSync,
		RunAt:             now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, database.Conn().Create(fresh).Error)

	count, err := ResetStuckProcessingIntegrationRequests(15 * time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "expected only the stale request to be reset")

	reloadedStale := &IntegrationRequest{}
	require.NoError(t, database.Conn().Where("id = ?", stale.ID).First(reloadedStale).Error)
	assert.Equal(t, IntegrationRequestStatePending, reloadedStale.State)

	reloadedFresh := &IntegrationRequest{}
	require.NoError(t, database.Conn().Where("id = ?", fresh.ID).First(reloadedFresh).Error)
	assert.Equal(t, IntegrationRequestStateProcessing, reloadedFresh.State)
}
