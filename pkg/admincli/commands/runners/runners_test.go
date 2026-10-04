package runners

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	assert.Contains(t, stdout.String(), "2026-10-02T19:00:00Z")
	assert.NotContains(t, stdout.String(), "ago")
}

func TestListRunnersPassesFleetAndFilters(t *testing.T) {
	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour).Truncate(time.Second)
	seen := now.Add(-5 * time.Minute).Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/admin/api/installation/fleets/aws-large-amd64/runners", r.URL.Path)
		assert.Equal(t, []string{"idle", "busy"}, r.URL.Query()["states"])
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(
			w,
			`{"runners":[{"id":"runner-1","state":"idle","createdAt":%q,"lastSeenAt":%q}]}`,
			created.Format(time.RFC3339),
			seen.Format(time.RFC3339),
		)
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
	assert.Contains(t, stdout.String(), "2h ago")
	assert.Contains(t, stdout.String(), "5m ago")
	assert.NotContains(t, stdout.String(), created.Format(time.RFC3339))
	assert.NotContains(t, stdout.String(), seen.Format(time.RFC3339))
}

func TestDescribeRunnerUsesFleetAndRunnerIDs(t *testing.T) {
	times := sampleRunnerTimes()
	server := runnerServer(t, http.MethodGet, times)
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&describeCommand{runnerCommand: runnerCommand{
		FleetID:  "aws-large-amd64",
		RunnerID: "runner-1",
	}}).Execute(ctx)

	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "ID: runner-1")
	assert.Contains(t, output, "Registered: 6h ago\n")
	assert.Contains(t, output, "Last seen: 5m ago\n")
	assert.Contains(t, output, "Created: 2h ago\n")
	assert.Contains(t, output, "Updated: 3d ago\n")
	assert.Contains(t, output, "Terminated: -\n")
	assert.NotContains(t, output, times.created.Format(time.RFC3339))
}

func TestDescribeRunnerJSONKeepsAbsoluteTimestamp(t *testing.T) {
	created := time.Now().UTC().Truncate(time.Second)
	stamp := created.Format(time.RFC3339)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"runner":{"id":"runner-1","createdAt":%q}}`, stamp)
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "json")
	err := (&describeCommand{runnerCommand: runnerCommand{
		FleetID:  "aws-large-amd64",
		RunnerID: "runner-1",
	}}).Execute(ctx)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), stamp)
	assert.NotContains(t, stdout.String(), "ago")
}

func TestDeleteRunnerUsesFleetAndRunnerIDs(t *testing.T) {
	times := sampleRunnerTimes()
	server := runnerServer(t, http.MethodDelete, times)
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&deleteCommand{runnerCommand: runnerCommand{
		FleetID:  "aws-large-amd64",
		RunnerID: "runner-1",
	}}).Execute(ctx)

	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "ID: runner-1")
	assert.Contains(t, output, "Created: 2h ago\n")
	assert.NotContains(t, output, times.created.Format(time.RFC3339))
}

type runnerTimes struct {
	registered time.Time
	seen       time.Time
	created    time.Time
	updated    time.Time
}

func sampleRunnerTimes() runnerTimes {
	now := time.Now().UTC()
	return runnerTimes{
		registered: now.Add(-6 * time.Hour).Truncate(time.Second),
		seen:       now.Add(-5 * time.Minute).Truncate(time.Second),
		created:    now.Add(-2 * time.Hour).Truncate(time.Second),
		updated:    now.Add(-72 * time.Hour).Truncate(time.Second),
	}
}

func runnerServer(t *testing.T, method string, times runnerTimes) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, method, r.Method)
		assert.Equal(
			t,
			"/admin/api/installation/fleets/aws-large-amd64/runners/runner-1",
			r.URL.Path,
		)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(
			w,
			`{"runner":{"id":"runner-1","fleetId":"aws-large-amd64","state":"idle","registeredAt":%q,"lastSeenAt":%q,"createdAt":%q,"updatedAt":%q}}`,
			times.registered.Format(time.RFC3339),
			times.seen.Format(time.RFC3339),
			times.created.Format(time.RFC3339),
			times.updated.Format(time.RFC3339),
		)
	}))
}
