package workers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/dockerhub"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/impl"
)

// Test__DockerHubRefreshLoopDoesNotAccumulate reproduces issue #5386.
//
// DockerHub drives its token refresh through ScheduleActionCall("refreshAccessToken").
// Re-running Sync (which happens on every integration sync/edit/capability update,
// and on the recurring refresh) must not leave behind an extra scheduled refresh -
// the loop must keep at most one pending request per installation.
//
// Before the ScheduleActionCall de-duplication fix this FAILS (the pending count
// climbs past 1, as it did in production with 202 orphaned chains). After the fix
// it PASSES, with the single pending request being the refreshAccessToken hook.
func Test__DockerHubRefreshLoopDoesNotAccumulate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dockerhub"] = &dockerhub.DockerHub{}
	integration := createDockerHubIntegration(t, r)

	syncOnce := func() {
		ctx := workercontexts.NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry, nil)
		httpCtx := &supportcontexts.HTTPContext{
			Responses: []*http.Response{dockerHubTokenResponse(t), dockerHubRepositoriesResponse()},
		}

		require.NoError(t, (&dockerhub.DockerHub{}).Sync(core.SyncContext{
			HTTP:          httpCtx,
			Integration:   ctx,
			Configuration: integration.Configuration.Data(),
		}))
	}

	//
	// The initial sync seeds a single scheduled refresh.
	//
	syncOnce()
	pending := pendingActionRequests(t, integration.ID, "refreshAccessToken")
	require.Len(t, pending, 1, "initial sync should schedule exactly one refresh")
	require.Equal(t, models.IntegrationRequestTypeInvokeAction, pending[0].Type)

	//
	// Subsequent syncs must reuse that single scheduled refresh, not stack up new
	// self-perpetuating chains (issue #5386).
	//
	for i := 0; i < 4; i++ {
		syncOnce()
		require.Len(t, pendingActionRequests(t, integration.ID, "refreshAccessToken"), 1,
			"DockerHub refresh loop must keep a single pending request after re-sync %d (issue #5386)", i+1)
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

func createDockerHubIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	//
	// accessToken is a Sensitive config field, so it is stored as
	// base64(encrypt(value, associatedData=installationID)).
	//
	integrationID := uuid.New()
	encryptedToken, err := r.Encryptor.Encrypt(context.Background(), []byte("pat"), []byte(integrationID.String()))
	require.NoError(t, err)

	integration, err := models.CreateIntegration(
		integrationID,
		r.Organization.ID,
		"dockerhub",
		support.RandomName("integration"),
		map[string]any{
			"username":    "superplane",
			"accessToken": base64.StdEncoding.EncodeToString(encryptedToken),
		},
	)
	require.NoError(t, err)

	return integration
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

func dockerHubTokenResponse(t *testing.T) *http.Response {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(map[string]any{"exp": time.Now().Add(10 * time.Minute).Unix()})
	require.NoError(t, err)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	jwt := header + "." + payload + ".signature"

	body, err := json.Marshal(map[string]any{"access_token": jwt})
	require.NoError(t, err)

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func dockerHubRepositoriesResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`
			{
				"next": null,
				"results": [
					{"name": "demo", "namespace": "superplane"}
				]
			}
		`)),
	}
}
