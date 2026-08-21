package devbroker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

// realHTTP satisfies the platform's HTTP context with the standard client, so
// the broker client below talks to the test server over a real connection.
type realHTTP struct{}

func (realHTTP) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// TestContract_SuperPlaneBrokerClient drives this broker with the same client
// Runner components use, so a response shape SuperPlane rejects fails here
// rather than at runtime. It caught the create endpoint answering 200 where the
// client requires 201.
func TestContract_SuperPlaneBrokerClient(t *testing.T) {
	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	t.Setenv("TASK_BROKER_BASE_URL", srv.URL)
	t.Setenv("TASK_BROKER_AUTH_TOKEN", testAuthToken)

	client, err := runner.NewBrokerClient(realHTTP{})
	require.NoError(t, err)

	taskID, err := client.CreateTask(runner.CreateTaskParams{
		MachineType: "e1-large-amd64",
		Commands:    []runner.BrokerCommand{{Command: "echo $GREETING"}},
		Environment: []runner.BrokerEnvironmentVariable{{Name: "GREETING", Value: "from-superplane"}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task := waitForTerminalState(t, client, taskID)
	assert.Equal(t, "succeeded", task.Status)
	assert.Contains(t, task.Output, "from-superplane")

	active, err := client.ListActiveTasks()
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "e1-large-amd64", active[0].FleetID)
}

func TestContract_CancelTask(t *testing.T) {
	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	t.Setenv("TASK_BROKER_BASE_URL", srv.URL)
	t.Setenv("TASK_BROKER_AUTH_TOKEN", testAuthToken)

	client, err := runner.NewBrokerClient(realHTTP{})
	require.NoError(t, err)

	taskID, err := client.CreateTask(runner.CreateTaskParams{
		MachineType: "e1-large-amd64",
		Commands:    []runner.BrokerCommand{{Command: "sleep 30"}},
	})
	require.NoError(t, err)

	require.NoError(t, client.CancelTask(taskID))

	task := waitForTerminalState(t, client, taskID)
	assert.Equal(t, "canceled", task.Status)
}

func waitForTerminalState(t *testing.T, client *runner.BrokerClient, taskID string) *runner.Task {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		task, err := client.FetchTaskStatus(taskID)
		require.NoError(t, err)
		if task.IsInTerminalState() {
			return task
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("task %s never reached a terminal state", taskID)
	return nil
}
