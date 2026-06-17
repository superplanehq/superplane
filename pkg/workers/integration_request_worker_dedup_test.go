package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

// Test__IntegrationRequestWorker_SelfReschedulingSyncDoesNotAccumulate reproduces
// issue #5386 with a generic, integration-agnostic loop: an integration whose Sync
// reschedules a recurring action via ScheduleActionCall (e.g. a token refresh).
//
// Sync re-runs on create and on every integration edit/capability update, so each
// run must reuse the single scheduled action rather than stacking a new
// self-perpetuating chain. Before the ScheduleActionCall de-duplication fix this
// FAILS (the pending count climbs, as it did in production with hundreds of
// orphaned chains); after the fix it stays at one.
func Test__IntegrationRequestWorker_SelfReschedulingSyncDoesNotAccumulate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	dummy := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		OnSync: func(ctx core.SyncContext) error {
			return ctx.Integration.ScheduleActionCall("refresh", map[string]any{}, time.Second)
		},
	})
	r.Registry.Integrations["dummy"] = dummy

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	syncOnce := func() {
		ctx := workercontexts.NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry, nil)
		require.NoError(t, dummy.Sync(core.SyncContext{
			Integration:   ctx,
			Configuration: integration.Configuration.Data(),
		}))
	}

	//
	// The initial sync seeds a single scheduled action.
	//
	syncOnce()
	require.Len(t, pendingActionRequests(t, integration.ID, "refresh"), 1, "initial sync should schedule exactly one action")

	//
	// Subsequent syncs must reuse that single scheduled action, not stack up new
	// self-perpetuating chains (issue #5386).
	//
	for i := 0; i < 4; i++ {
		syncOnce()
		require.Len(t, pendingActionRequests(t, integration.ID, "refresh"), 1,
			"a self-rescheduling sync loop must keep a single pending request after re-sync %d (#5386)", i+1)
	}
}

// Test__IntegrationRequestWorker_ActionCallNoDuplicateOnRetry validates that a
// self-rescheduling action loop cannot duplicate when a leased request is retried
// (e.g. after a worker crash or a phase-3 failure). Completing the matching pending
// request and creating the successor happen atomically, so reprocessing the
// already-completed parent is a no-op (#5386).
func Test__IntegrationRequestWorker_ActionCallNoDuplicateOnRetry(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewIntegrationRequestWorker(r.Encryptor, r.Registry, nil, "http://localhost:8000", "http://localhost:8000")

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Hooks: []core.Hook{{Name: "refresh", Parameters: []configuration.Field{}}},
		HandleHook: func(ctx core.IntegrationHookContext) error {
			return ctx.Integration.ScheduleActionCall("refresh", map[string]any{}, time.Second)
		},
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "dummy", support.RandomName("integration"), nil)
	require.NoError(t, err)

	runAt := time.Now().Add(-time.Second)
	require.NoError(t, integration.CreateActionRequest(database.Conn(), "refresh", map[string]any{}, &runAt))
	requests, err := integration.ListRequests(models.IntegrationRequestTypeInvokeAction)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	request := requests[0]

	//
	// First pass: completes the request and atomically schedules its successor.
	//
	require.NoError(t, worker.LockAndProcessRequest(request))
	require.Len(t, pendingActionRequests(t, integration.ID, "refresh"), 1,
		"exactly one successor should be pending after processing")

	//
	// Reprocess the very same leased request, simulating a lease retry after a
	// crash. The already-completed parent must not spawn a second chain.
	//
	require.NoError(t, worker.LockAndProcessRequest(request))
	require.Len(t, pendingActionRequests(t, integration.ID, "refresh"), 1,
		"a retried (already completed) request must not create a duplicate chain (#5386)")
}

func pendingActionRequests(t *testing.T, integrationID uuid.UUID, actionName string) []models.IntegrationRequest {
	t.Helper()

	var requests []models.IntegrationRequest
	require.NoError(t, database.Conn().
		Where("app_installation_id = ? AND state = ? AND type = ?",
			integrationID, models.IntegrationRequestStatePending, models.IntegrationRequestTypeInvokeAction).
		Where("spec->'invoke_action'->>'action_name' = ?", actionName).
		Find(&requests).Error)
	return requests
}
