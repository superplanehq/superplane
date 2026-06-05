package broker

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/wsrunner"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func queueTask(t *testing.T, ctx context.Context, st interface {
	CreateTask(context.Context, *models.Task) error
}, fleetID, cmd string) {
	t.Helper()
	require.NoError(t, st.CreateTask(ctx, &models.Task{
		ID:        uuid.NewString(),
		FleetID:   fleetID,
		Commands:  []string{cmd},
		Status:    models.StatusQueued,
		CreatedAt: time.Now().UTC(),
	}))
}

func oneShotBrokerSetup(t *testing.T) (*httptest.Server, interface {
	CreateFleet(context.Context, *brokermodels.Fleet) error
	CreateTask(context.Context, *models.Task) error
	ListActiveTasks(context.Context) ([]*models.Task, error)
}) {
	t.Helper()
	st, cleanup := testdb.Open(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	require.NoError(t, st.CreateFleet(ctx, &brokermodels.Fleet{
		ID:          "fleet-1",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   time.Now().UTC(),
	}))

	srv := &Server{Store: st, TaskNotify: NewWaitHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	t.Cleanup(ts.Close)
	return ts, st
}

func dialWS(t *testing.T, ts *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + ts.URL[4:] + "/v1/runners/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, map[string][]string{
		"Authorization": {"Bearer token"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestOneShotRunnerReceivesExactlyOneTask verifies that a runner sending one_shot:true
// in its Hello message receives exactly one task even when multiple tasks are queued,
// and that the broker closes the connection without claiming a second task.
func TestOneShotRunnerReceivesExactlyOneTask(t *testing.T) {
	ts, st := oneShotBrokerSetup(t)
	ctx := context.Background()

	queueTask(t, ctx, st, "fleet-1", "echo task-1")
	queueTask(t, ctx, st, "fleet-1", "echo task-2")

	conn := dialWS(t, ts)

	require.NoError(t, conn.WriteJSON(wsrunner.Hello{
		Type:         wsrunner.TypeHello,
		RunnerID:     "i-oneshot",
		FleetID:      "fleet-1",
		LeaseSeconds: 300,
		OneShot:      true,
	}))

	// Receive first task.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var taskMsg wsrunner.Task
	require.NoError(t, conn.ReadJSON(&taskMsg))
	require.Equal(t, wsrunner.TypeTask, taskMsg.Type)
	require.NotNil(t, taskMsg.Task)

	// Complete it.
	require.NoError(t, conn.WriteJSON(wsrunner.Complete{
		Type:     wsrunner.TypeComplete,
		TaskID:   taskMsg.Task.ID,
		RunnerID: "i-oneshot",
		ExitCode: 0,
	}))

	// Expect ack.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var ack wsrunner.Ack
	require.NoError(t, conn.ReadJSON(&ack))
	require.Equal(t, wsrunner.TypeAck, ack.Type)

	// Broker must close the connection after one_shot — next read should error.
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err := conn.ReadMessage()
	require.Error(t, err, "broker should have closed the WS after one_shot task")

	// Second task must still be queued (not claimed by this runner).
	tasks, err := st.ListActiveTasks(ctx)
	require.NoError(t, err)
	queued := 0
	for _, task := range tasks {
		if string(task.Status) == "queued" {
			queued++
		}
	}
	require.Equal(t, 1, queued, "second task should remain queued, not claimed by the one-shot runner")
}

// TestMultiShotRunnerReceivesMultipleTasks verifies that a normal (non-one-shot) runner
// continues receiving tasks after completing one.
func TestMultiShotRunnerReceivesMultipleTasks(t *testing.T) {
	ts, st := oneShotBrokerSetup(t)
	ctx := context.Background()

	queueTask(t, ctx, st, "fleet-1", "echo task-1")
	queueTask(t, ctx, st, "fleet-1", "echo task-2")

	conn := dialWS(t, ts)

	require.NoError(t, conn.WriteJSON(wsrunner.Hello{
		Type:         wsrunner.TypeHello,
		RunnerID:     "i-multishot",
		FleetID:      "fleet-1",
		LeaseSeconds: 300,
		OneShot:      false,
	}))

	var received atomic.Int32
	for i := 0; i < 2; i++ {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var taskMsg wsrunner.Task
		require.NoError(t, conn.ReadJSON(&taskMsg))
		require.Equal(t, wsrunner.TypeTask, taskMsg.Type)
		received.Add(1)

		require.NoError(t, conn.WriteJSON(wsrunner.Complete{
			Type:     wsrunner.TypeComplete,
			TaskID:   taskMsg.Task.ID,
			RunnerID: "i-multishot",
			ExitCode: 0,
		}))
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var ack wsrunner.Ack
		require.NoError(t, conn.ReadJSON(&ack))
		require.Equal(t, wsrunner.TypeAck, ack.Type)
	}
	require.Equal(t, int32(2), received.Load(), "multi-shot runner should handle both tasks")
}
