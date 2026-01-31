package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__IntegrationRequestWorker_Sync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var syncCalled bool
	r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			ctx.Integration.Ready()
			syncCalled = true
			return nil
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	//
	// Create the integration sync request
	//
	runAt := time.Now().Add(-time.Second)
	require.NoError(t, integration.CreateSyncRequest(database.Conn(), &runAt))
	requests, err := integration.ListRequests(models.IntegrationRequestTypeSync)
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
	request, err = integration.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationRequestStateCompleted, request.State)
	assert.True(t, syncCalled)
}

func Test__IntegrationRequestWorker_SyncError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var syncCalled bool
	r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			syncCalled = true
			return errors.New("sync failed")
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	//
	// Create the integration sync request
	//
	runAt := time.Now().Add(-time.Second)
	require.NoError(t, integration.CreateSyncRequest(database.Conn(), &runAt))
	requests, err := integration.ListRequests(models.IntegrationRequestTypeSync)
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
	request, err = integration.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationRequestStateCompleted, request.State)
	assert.True(t, syncCalled)

	integration, err = models.FindIntegration(r.Organization.ID, integration.ID)
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationStateError, integration.State)
	assert.Contains(t, integration.StateDescription, "Sync failed: sync failed")
}

func Test__AppInstallationRequestWorker_InvokeAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var actionCalled bool
	r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
		Actions: []core.Action{
			{
				Name:       "test",
				Parameters: []configuration.Field{},
			},
		},
		HandleAction: func(ctx core.IntegrationActionContext) error {
			actionCalled = true
			return nil
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	//
	// Create the integration sync request
	//
	runAt := time.Now().Add(-time.Second)
	require.NoError(t, integration.CreateActionRequest(database.Conn(), "test", nil, &runAt))
	requests, err := integration.ListRequests(models.IntegrationRequestTypeInvokeAction)
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
	request, err = integration.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationRequestStateCompleted, request.State)
	assert.True(t, actionCalled)
}

func Test__AppInstallationRequestWorker_InvokeActionError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var actionCalled bool
	r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
		Actions: []core.Action{
			{
				Name:       "test",
				Parameters: []configuration.Field{},
			},
		},
		HandleAction: func(ctx core.IntegrationActionContext) error {
			actionCalled = true
			return errors.New("action failed")
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	//
	// Create the integration sync request
	//
	runAt := time.Now().Add(-time.Second)
	require.NoError(t, integration.CreateActionRequest(database.Conn(), "test", nil, &runAt))
	requests, err := integration.ListRequests(models.IntegrationRequestTypeInvokeAction)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	request := &requests[0]

	//
	// Process request
	//
	require.NoError(t, worker.LockAndProcessRequest(*request))

	//
	// Reload request, verify it was completed, even though the action failed.
	//
	request, err = integration.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationRequestStateCompleted, request.State)
	assert.True(t, actionCalled)
}
