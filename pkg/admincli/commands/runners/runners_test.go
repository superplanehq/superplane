package runners

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clitest "github.com/superplanehq/superplane/test/support/cli"
)

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
