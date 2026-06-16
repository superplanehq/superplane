package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func Test__LeaseIntegrationRequest(t *testing.T) {
	const lease = 5 * time.Minute

	newDueRequest := func(t *testing.T) *IntegrationRequest {
		require.NoError(t, database.TruncateTables())

		organization, err := CreateOrganization("org-"+uuid.NewString(), "")
		require.NoError(t, err)
		integration, err := CreateIntegration(uuid.New(), organization.ID, "dummy", "integration-"+uuid.NewString(), nil)
		require.NoError(t, err)

		runAt := time.Now().Add(-time.Second)
		require.NoError(t, integration.CreateSyncRequest(database.Conn(), &runAt))
		request, err := FindPendingRequestForIntegration(database.Conn(), integration.ID)
		require.NoError(t, err)
		return request
	}

	t.Run("leasing a due request excludes it from the poll", func(t *testing.T) {
		request := newDueRequest(t)

		before := time.Now()
		leased, err := LeaseIntegrationRequest(database.Conn(), request.ID, lease)
		require.NoError(t, err)

		//
		// run_at is pushed roughly a lease into the future, so the request drops
		// out of the due-pending poll while it is being processed.
		//
		assert.True(t, leased.RunAt.After(before.Add(lease-time.Minute)),
			"expected run_at to be pushed ~lease into the future")

		listed, err := ListIntegrationRequests()
		require.NoError(t, err)
		assert.Empty(t, listed, "a leased request must not be listed as due")
	})

	t.Run("a second lease attempt while leased returns not found", func(t *testing.T) {
		request := newDueRequest(t)

		_, err := LeaseIntegrationRequest(database.Conn(), request.ID, lease)
		require.NoError(t, err)

		_, err = LeaseIntegrationRequest(database.Conn(), request.ID, lease)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound,
			"a request that is already leased (future run_at) must not be leased again")
	})

	t.Run("an expired lease becomes due again", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		organization, err := CreateOrganization("org-"+uuid.NewString(), "")
		require.NoError(t, err)
		integration, err := CreateIntegration(uuid.New(), organization.ID, "dummy", "integration-"+uuid.NewString(), nil)
		require.NoError(t, err)

		//
		// Simulate a worker that leased the request and then died: the row stays
		// pending with a run_at in the past (lease expired). It must show up as due
		// again so it gets retried, with no separate reset mechanism.
		//
		now := time.Now()
		expired := &IntegrationRequest{
			ID:                uuid.New(),
			AppInstallationID: integration.ID,
			State:             IntegrationRequestStatePending,
			Type:              IntegrationRequestTypeSync,
			RunAt:             now.Add(-time.Second),
			CreatedAt:         now.Add(-time.Hour),
			UpdatedAt:         now.Add(-time.Hour),
		}
		require.NoError(t, database.Conn().Create(expired).Error)

		listed, err := ListIntegrationRequests()
		require.NoError(t, err)
		require.Len(t, listed, 1, "an expired lease must be listed as due again")
		assert.Equal(t, expired.ID, listed[0].ID)

		leased, err := LeaseIntegrationRequest(database.Conn(), expired.ID, lease)
		require.NoError(t, err, "an expired lease must be re-leasable")
		assert.True(t, leased.RunAt.After(now), "re-leasing pushes run_at into the future again")
	})
}
