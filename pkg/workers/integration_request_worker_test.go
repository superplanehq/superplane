package workers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

func Test__IntegrationRequestWorker_Sync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var syncCalled bool
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
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

// Test__IntegrationRequestWorker_SyncRewritesSecret drives the self-perpetuating
// token-refresh loop behind issue #5386: each sync rewrites the secret and
// reschedules itself. It verifies the loop keeps rewriting a single row (rather
// than creating new ones) and that the background loop self-perpetuates.
func Test__IntegrationRequestWorker_SyncRewritesSecret(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	encryptor := crypto.NewAESGCMEncryptor([]byte("0123456789abcdef0123456789abcdef"))
	worker := NewIntegrationRequestWorker(encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	const (
		secretName = "access_token"
		cycles     = 5
	)

	var token int
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			token++
			if err := ctx.Integration.SetSecret(secretName, []byte(fmt.Sprintf("token-%d", token))); err != nil {
				return err
			}
			return ctx.Integration.ScheduleResync(time.Second)
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, integration.CreateSyncRequest(database.Conn(), &now))

	ciphertexts := map[string]struct{}{}
	updatedAts := make([]time.Time, 0, cycles)
	for i := 0; i < cycles; i++ {
		request, err := models.FindPendingRequestForIntegration(database.Conn(), integration.ID)
		require.NoError(t, err, "expected a pending sync request before cycle %d", i)
		require.NoError(t, worker.LockAndProcessRequest(*request))

		var secret models.IntegrationSecret
		require.NoError(t, database.Conn().
			Where("installation_id = ? AND name = ?", integration.ID, secretName).
			First(&secret).Error)
		ciphertexts[string(secret.Value)] = struct{}{}
		updatedAts = append(updatedAts, *secret.UpdatedAt)
	}

	//
	// The loop rewrites a single row rather than creating new ones.
	//
	var rowCount int64
	require.NoError(t, database.Conn().Model(&models.IntegrationSecret{}).
		Where("installation_id = ? AND name = ?", integration.ID, secretName).
		Count(&rowCount).Error)
	assert.Equal(t, int64(1), rowCount, "expected the same secret row to be updated, not duplicated")

	//
	// The background loop self-perpetuated: one completed sync per cycle, plus
	// one pending request scheduled for the next round.
	//
	requests, err := integration.ListRequests(models.IntegrationRequestTypeSync)
	require.NoError(t, err)
	var completed, pending int
	for _, request := range requests {
		switch request.State {
		case models.IntegrationRequestStateCompleted:
			completed++
		case models.IntegrationRequestStatePending:
			pending++
		}
	}
	assert.Equal(t, cycles, completed, "expected one completed sync request per cycle")
	assert.Equal(t, 1, pending, "expected exactly one pending sync request scheduled for the next round")

	//
	// Each cycle changed the value, so each write produced a distinct ciphertext
	// and advanced updated_at.
	//
	assert.Len(t, ciphertexts, cycles, "expected each changed write to produce a distinct ciphertext")
	for i := 1; i < len(updatedAts); i++ {
		assert.True(t, updatedAts[i].After(updatedAts[i-1]),
			"expected updated_at to advance on every cycle (cycle %d)", i)
	}
}

// Test__IntegrationRequestWorker_ClaimsBeforeProcessing covers the Part 2 fix:
// the request is claimed (pending -> processing) in a short transaction before the
// external work runs, so the work happens outside any DB transaction and the
// in-flight request is no longer re-listed by the poll loop.
func Test__IntegrationRequestWorker_ClaimsBeforeProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	var installationID uuid.UUID
	var stateDuringSync string
	var listedDuringSync int
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			var request models.IntegrationRequest
			require.NoError(t, database.Conn().
				Where("app_installation_id = ?", installationID).
				First(&request).Error)
			stateDuringSync = request.State

			listed, err := models.ListIntegrationRequests()
			require.NoError(t, err)
			for _, listedRequest := range listed {
				if listedRequest.AppInstallationID == installationID {
					listedDuringSync++
				}
			}
			return nil
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)
	installationID = integration.ID

	runAt := time.Now().Add(-time.Second)
	require.NoError(t, integration.CreateSyncRequest(database.Conn(), &runAt))
	requests, err := integration.ListRequests(models.IntegrationRequestTypeSync)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	request := &requests[0]

	require.NoError(t, worker.LockAndProcessRequest(*request))

	assert.Equal(t, models.IntegrationRequestStateProcessing, stateDuringSync,
		"request should be in processing state while the external work runs")
	assert.Equal(t, 0, listedDuringSync,
		"an in-flight (processing) request must not be re-listed by the poll loop")

	request, err = integration.GetRequest(request.ID.String())
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationRequestStateCompleted, request.State)
}

func Test__IntegrationRequestWorker_SyncError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var syncCalled bool
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
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

func Test__AppInstallationRequestWorker_InvokeHook(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var hookCalled bool
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Hooks: []core.Hook{
			{
				Name:       "test",
				Parameters: []configuration.Field{},
			},
		},
		HandleHook: func(ctx core.IntegrationHookContext) error {
			hookCalled = true
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
	assert.True(t, hookCalled)
}

func Test__AppInstallationRequestWorker_InvokeHookError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	//
	// Register a dummy application and install it.
	//
	var hookCalled bool
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Hooks: []core.Hook{
			{
				Name:       "test",
				Parameters: []configuration.Field{},
			},
		},
		HandleHook: func(ctx core.IntegrationHookContext) error {
			hookCalled = true
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
	assert.True(t, hookCalled)
}
