package contexts

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AppInstallationContext_ScheduleResync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create app installation
	//
	installation, err := models.CreateAppInstallation(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("installation"),
		map[string]any{},
	)
	require.NoError(t, err)

	ctx := NewAppInstallationContext(database.Conn(), nil, installation, r.Encryptor, r.Registry)

	t.Run("rejects short interval", func(t *testing.T) {
		err = ctx.ScheduleResync(500 * time.Millisecond)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "interval must be bigger than 1s")
	})

	t.Run("completes previous request on new request", func(t *testing.T) {
		//
		// Create previous request
		//
		now := time.Now()
		require.NoError(t, installation.CreateSyncRequest(database.Conn(), &now))
		requests, err := installation.ListRequests(models.AppInstallationRequestTypeSync)
		require.NoError(t, err)
		require.Len(t, requests, 1)
		previousRequest := &requests[0]

		//
		// Schedule new sync request.
		//
		require.NoError(t, ctx.ScheduleResync(2*time.Second))

		//
		// Verify previous request was completed.
		//
		previousRequest, err = installation.GetRequest(previousRequest.ID.String())
		require.NoError(t, err)
		require.Equal(t, models.AppInstallationRequestStateCompleted, previousRequest.State)

		//
		// Verify new one was created
		//
		requests, err = installation.ListRequests(models.AppInstallationRequestTypeSync)
		require.NoError(t, err)
		require.Len(t, requests, 2)
		newRequestIndex := slices.IndexFunc(requests, func(r models.AppInstallationRequest) bool { return r.ID.String() != previousRequest.ID.String() })
		newRequest := requests[newRequestIndex]
		require.Equal(t, models.AppInstallationRequestStatePending, newRequest.State)
	})
}
