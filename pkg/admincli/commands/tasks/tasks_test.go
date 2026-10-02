package tasks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clitest "github.com/superplanehq/superplane/test/support/cli"
)

func TestListTasksPassesFleetAndFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/admin/api/installation/fleets/aws-large-amd64/tasks", r.URL.Path)
		assert.Equal(t, []string{"running", "failed"}, r.URL.Query()["states"])
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"tasks":[{"id":"task-1","state":"running"}]}`)
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&listCommand{
		FleetID: "aws-large-amd64",
		States:  []string{"running", "failed"},
		Limit:   50,
	}).Execute(ctx)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "task-1")
}

func TestListTasksRejectsInvalidState(t *testing.T) {
	ctx, _ := clitest.NewCommandContext(t, nil, "text")
	err := (&listCommand{FleetID: "fleet", States: []string{"unknown"}, Limit: 200}).Execute(ctx)
	require.EqualError(t, err, `invalid task state "unknown"`)
}
