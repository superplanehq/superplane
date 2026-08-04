package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBroadcastToWorkflowUnregistersFullClientWithoutDeadlocking(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:        hub,
		send:       make(chan []byte, 1),
		Done:       make(chan struct{}),
		workflowID: "workflow",
	}
	client.send <- []byte("existing")

	hub.registerClient(client)

	done := make(chan struct{})
	go func() {
		hub.BroadcastToWorkflow("workflow", []byte("msg"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broadcast blocked while unregistering a full client")
	}

	require.Equal(t, 0, hub.WorkflowSubscriberCount("workflow"))
}
