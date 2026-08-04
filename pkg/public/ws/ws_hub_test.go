package ws

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient builds a client without a websocket connection. None of the
// broadcast paths touch conn, and a client with an unbuffered send channel and
// no reader always takes the "buffer full" branch, which is what these tests
// exercise.
func newTestClient(hub *Hub, workflowID string, sendBuffer int) *Client {
	return &Client{
		hub:        hub,
		send:       make(chan []byte, sendBuffer),
		Done:       make(chan struct{}),
		workflowID: workflowID,
	}
}

// registerTestClients registers clients and waits for the hub loop to pick them
// up, since sending on the register channel only hands the client over.
func registerTestClients(t *testing.T, hub *Hub, clients ...*Client) {
	t.Helper()

	expected := map[string]int{}
	for _, client := range clients {
		hub.register <- client
		expected[client.workflowID]++
	}

	require.Eventually(t, func() bool {
		for workflowID, count := range expected {
			if hub.WorkflowSubscriberCount(workflowID) != count {
				return false
			}
		}
		return true
	}, time.Second, time.Millisecond, "clients were never registered")
}

// requireNoReturnAfterTimeout fails if fn has not returned in time. A hang here
// is the failure being guarded against, not a slow test: evicting a client from
// inside a broadcast used to take the write lock while the read lock was still
// held, and sync.RWMutex is not reentrant.
func requireNoDeadlock(t *testing.T, fn func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("broadcast never returned: the hub deadlocked while evicting a stalled client")
	}
}

func TestBroadcastToWorkflowDoesNotDeadlockOnFullClientBuffer(t *testing.T) {
	hub := NewHub()
	hub.Run()

	registerTestClients(t, hub, newTestClient(hub, "workflow-1", 0))

	requireNoDeadlock(t, func() {
		hub.BroadcastToWorkflow("workflow-1", []byte("tick"))
	})
}

func TestBroadcastAllDoesNotDeadlockOnFullClientBuffer(t *testing.T) {
	hub := NewHub()
	hub.Run()

	registerTestClients(t, hub, newTestClient(hub, "workflow-1", 0))

	requireNoDeadlock(t, func() {
		hub.BroadcastAll([]byte("tick"))
	})
}

func TestBroadcastToWorkflowDropsStalledClientAndKeepsHealthyOnes(t *testing.T) {
	hub := NewHub()
	hub.Run()

	healthy := newTestClient(hub, "workflow-1", 1)
	stalled := newTestClient(hub, "workflow-1", 0)
	registerTestClients(t, hub, healthy, stalled)

	requireNoDeadlock(t, func() {
		hub.BroadcastToWorkflow("workflow-1", []byte("tick"))
	})

	assert.Equal(t, []byte("tick"), <-healthy.send)
	assert.Equal(t, 1, hub.WorkflowSubscriberCount("workflow-1"))

	hub.mutex.RLock()
	defer hub.mutex.RUnlock()
	assert.True(t, hub.clients[healthy], "healthy client should stay registered")
	assert.False(t, hub.clients[stalled], "stalled client should have been dropped")
}

func TestBroadcastToWorkflowIgnoresOtherWorkflows(t *testing.T) {
	hub := NewHub()
	hub.Run()

	subscriber := newTestClient(hub, "workflow-1", 1)
	other := newTestClient(hub, "workflow-2", 1)
	registerTestClients(t, hub, subscriber, other)

	hub.BroadcastToWorkflow("workflow-1", []byte("tick"))

	assert.Equal(t, []byte("tick"), <-subscriber.send)
	assert.Empty(t, other.send, "clients watching another workflow should not receive the message")
}

// Two broadcasters can race to evict the same stalled client. Eviction closes
// the client's send channel, so it has to stay idempotent or the second one
// panics.
func TestConcurrentBroadcastsDropSameStalledClientSafely(t *testing.T) {
	hub := NewHub()
	hub.Run()

	registerTestClients(t, hub, newTestClient(hub, "workflow-1", 0))

	var broadcasts sync.WaitGroup
	for range 8 {
		broadcasts.Add(1)
		go func() {
			defer broadcasts.Done()
			hub.BroadcastToWorkflow("workflow-1", []byte("tick"))
		}()
	}

	requireNoDeadlock(t, broadcasts.Wait)
	assert.Equal(t, 0, hub.WorkflowSubscriberCount("workflow-1"))
}

func TestUnregisterClientIsIdempotent(t *testing.T) {
	hub := NewHub()
	client := newTestClient(hub, "workflow-1", 0)

	hub.mutex.Lock()
	hub.clients[client] = true
	hub.workflowSubscriptions["workflow-1"] = map[*Client]bool{client: true}
	hub.mutex.Unlock()

	hub.unregisterClient(client)

	assert.NotPanics(t, func() {
		hub.unregisterClient(client)
	}, "unregistering twice must not close the send channel twice")

	assert.Equal(t, 0, hub.WorkflowSubscriberCount("workflow-1"))
}
