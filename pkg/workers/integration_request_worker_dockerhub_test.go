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
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/dockerhub"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

// Test__DockerHubRefreshLoopDoesNotAccumulate reproduces issue #5386.
//
// The DockerHub token-refresh loop must keep at most one pending integration
// request per installation. Running Sync again (which happens on every
// integration sync/edit/capability update, and on the recurring refresh) must
// not leave behind an extra scheduled refresh.
//
// Today Sync reschedules via ScheduleActionCall, which - unlike ScheduleResync -
// never completes the existing pending request. So each Sync permanently adds a
// new, self-perpetuating refresh chain; in production this accumulated to 202
// orphaned chains hammering UPDATE app_installation_secrets.
//
// This test FAILS on the current code (pending count climbs past 1) and PASSES
// once the loop is switched to the deduplicated ScheduleResync.
func Test__DockerHubRefreshLoopDoesNotAccumulate(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dockerhub"] = &dockerhub.DockerHub{}

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

	pendingRequests := func() int64 {
		var count int64
		require.NoError(t, database.Conn().
			Model(&models.IntegrationRequest{}).
			Where("app_installation_id = ? AND state = ?", integration.ID, models.IntegrationRequestStatePending).
			Count(&count).Error)
		return count
	}

	//
	// The initial sync seeds a single scheduled refresh.
	//
	syncOnce()
	require.Equal(t, int64(1), pendingRequests(), "initial sync should schedule exactly one refresh")

	//
	// Subsequent syncs must reuse that single scheduled refresh, not stack up
	// new self-perpetuating chains (issue #5386).
	//
	for i := 0; i < 4; i++ {
		syncOnce()
		require.Equal(t, int64(1), pendingRequests(),
			"DockerHub refresh loop must keep a single pending request after re-sync %d (issue #5386)", i+1)
	}
}

// Test__DockerHubSyncSchedulesResyncRequest verifies the refresh loop is driven
// by a deduplicated sync request rather than a self-perpetuating action call,
// which is the fix for issue #5386.
func Test__DockerHubSyncSchedulesResyncRequest(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dockerhub"] = &dockerhub.DockerHub{}
	integration := createDockerHubIntegration(t, r)

	ctx := workercontexts.NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry, nil)
	httpCtx := &supportcontexts.HTTPContext{
		Responses: []*http.Response{dockerHubTokenResponse(t), dockerHubRepositoriesResponse()},
	}

	require.NoError(t, (&dockerhub.DockerHub{}).Sync(core.SyncContext{
		HTTP:          httpCtx,
		Integration:   ctx,
		Configuration: integration.Configuration.Data(),
	}))

	pending := pendingRequestsForIntegration(t, integration.ID)
	require.Len(t, pending, 1, "DockerHub sync should schedule exactly one refresh (#5386)")
	require.Equal(t, models.IntegrationRequestTypeSync, pending[0].Type,
		"DockerHub must reschedule its refresh as a deduplicated sync request, not an action call (#5386)")
}

// Test__DockerHubRefreshHookBridgesToResync verifies the backwards-compatibility
// bridge: a leftover self-perpetuating "refreshAccessToken" action request (one
// of the 202 orphaned chains in #5386) collapses onto the single deduplicated
// sync loop when it runs.
func Test__DockerHubRefreshHookBridgesToResync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["dockerhub"] = &dockerhub.DockerHub{}
	integration := createDockerHubIntegration(t, r)

	//
	// Simulate one of the orphaned action chains that drove the runaway loop.
	//
	runAt := time.Now()
	require.NoError(t, integration.CreateActionRequest(database.Conn(), "refreshAccessToken", map[string]any{}, &runAt))

	ctx := workercontexts.NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry, nil)
	httpCtx := &supportcontexts.HTTPContext{
		Responses: []*http.Response{dockerHubTokenResponse(t)},
	}

	require.NoError(t, (&dockerhub.DockerHub{}).HandleHook(core.IntegrationHookContext{
		Name:          "refreshAccessToken",
		HTTP:          httpCtx,
		Integration:   ctx,
		Configuration: integration.Configuration.Data(),
	}))

	pending := pendingRequestsForIntegration(t, integration.ID)
	require.Len(t, pending, 1, "the orphaned action chain must collapse onto a single request (#5386)")
	require.Equal(t, models.IntegrationRequestTypeSync, pending[0].Type,
		"the refresh hook must bridge the orphaned chain onto the sync resync loop (#5386)")
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

func pendingRequestsForIntegration(t *testing.T, integrationID uuid.UUID) []models.IntegrationRequest {
	t.Helper()

	var requests []models.IntegrationRequest
	require.NoError(t, database.Conn().
		Where("app_installation_id = ? AND state = ?", integrationID, models.IntegrationRequestStatePending).
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
