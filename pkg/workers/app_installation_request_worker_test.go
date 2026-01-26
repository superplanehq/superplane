package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__AppInstallationRequestWorker_Sync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppInstallationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var syncCalled bool
	r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
		ctx.AppInstallation.SetState("ready", "")
		syncCalled = true
		return nil
	})

	installation, err := models.CreateAppInstallation(uuid.New(), r.Organization.ID, "dummy", support.RandomName("installation"), nil)
	require.NoError(t, err)

	//
	// Create the app installation sync request
	//
	runAt := time.Now().Add(-time.Second)
	require.NoError(t, installation.CreateSyncRequest(database.Conn(), &runAt))
	requests, err := installation.ListRequests(models.AppInstallationRequestTypeSync)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	request := &requests[0]

	//
	// Lock and process request
	//
	err = worker.LockAndProcessRequest(*request)
	require.NoError(t, err)

	//
	// Reload request, verify it was completed, and sync was called
	//
	request, err = installation.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.AppInstallationRequestStateCompleted, request.State)
	assert.True(t, syncCalled)
}

func Test__AppInstallationRequestWorker_SyncError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppInstallationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var syncCalled bool
	r.Registry.Applications["dummy"] = support.NewDummyApplication(func(ctx core.SyncContext) error {
		syncCalled = true
		return errors.New("sync failed")
	})

	installation, err := models.CreateAppInstallation(uuid.New(), r.Organization.ID, "dummy", support.RandomName("installation"), nil)
	require.NoError(t, err)

	//
	// Create the app installation sync request
	//
	runAt := time.Now().Add(-time.Second)
	require.NoError(t, installation.CreateSyncRequest(database.Conn(), &runAt))
	requests, err := installation.ListRequests(models.AppInstallationRequestTypeSync)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	request := &requests[0]

	//
	// Process request
	//
	require.NoError(t, worker.LockAndProcessRequest(*request))

	//
	// Reload request, verify it was completed, and app installation was moved to error state.
	//
	request, err = installation.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.AppInstallationRequestStateCompleted, request.State)
	assert.True(t, syncCalled)

	installation, err = models.FindAppInstallation(r.Organization.ID, installation.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AppInstallationStateError, installation.State)
	assert.Contains(t, installation.StateDescription, "Sync failed: sync failed")
}
