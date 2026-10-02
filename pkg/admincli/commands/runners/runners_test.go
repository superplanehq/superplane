package runners

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	clitest "github.com/superplanehq/superplane/test/support/cli"
)

func TestCreateRunnerPassesLifecycleAndRendersRegistration(t *testing.T) {
	var request openapi_client.RunnersCreateRunnerBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/admin/api/installation/fleets/e1-large-arm64/runners", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"runner":{"id":"runner-1","fleetId":"e1-large-arm64","state":"pending","runnerVersion":"dev","ephemeral":true},
			"registrationToken":"registration-token",
			"registrationExpiresAt":"2026-10-02T19:00:00Z",
			"runnerApiUrl":"http://localhost:8000"
		}`)
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&createCommand{
		FleetID:        "e1-large-arm64",
		TaskID:         "task-1",
		IdempotencyKey: "request-1",
		Ephemeral:      true,
	}).Execute(ctx)

	require.NoError(t, err)
	assert.Equal(t, "task-1", request.GetTaskId())
	assert.Equal(t, "request-1", request.GetIdempotencyKey())
	assert.True(t, request.GetEphemeral())
	assert.Contains(t, stdout.String(), "runner-1")
	assert.Contains(t, stdout.String(), "registration-token")
	assert.Contains(t, stdout.String(), "http://localhost:8000")
}

func TestListRunnersPassesFleetAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/admin/api/installation/fleets/aws-large-amd64/runners", r.URL.Path)
		assert.Equal(t, []string{"idle", "busy"}, r.URL.Query()["states"])
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"runners":[{"id":"runner-1","state":"idle"}]}`)
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&listCommand{
		FleetID: "aws-large-amd64",
		States:  []string{"idle", "busy"},
		Limit:   25,
	}).Execute(ctx)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "runner-1")
}

func TestDescribeRunnerUsesFleetAndRunnerIDs(t *testing.T) {
	server := runnerServer(t, http.MethodGet)
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&describeCommand{runnerCommand: runnerCommand{
		FleetID:  "aws-large-amd64",
		RunnerID: "runner-1",
	}}).Execute(ctx)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "ID: runner-1")
}

func TestDeleteRunnerUsesFleetAndRunnerIDs(t *testing.T) {
	server := runnerServer(t, http.MethodDelete)
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&deleteCommand{runnerCommand: runnerCommand{
		FleetID:  "aws-large-amd64",
		RunnerID: "runner-1",
	}}).Execute(ctx)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "ID: runner-1")
}

func runnerServer(t *testing.T, method string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, method, r.Method)
		assert.Equal(
			t,
			"/admin/api/installation/fleets/aws-large-amd64/runners/runner-1",
			r.URL.Path,
		)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"runner":{"id":"runner-1","fleetId":"aws-large-amd64","state":"idle"}}`)
	}))
}
