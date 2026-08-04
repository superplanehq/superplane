package ws

import (
	"sync"
	"testing"
	"time"
)

func TestBroadcastToWorkflow_NoSubscribers(t *testing.T) {
	hub := NewHub()
	hub.Run()

	hub.BroadcastToWorkflow("missing-canvas", []byte(`{"ok":true}`))
}

func TestRegisterAndUnregisterClient(t *testing.T) {
	hub := NewHub()
	hub.Run()

	client := &Client{
		hub:        hub,
		send:       make(chan []byte, 1),
		Done:       make(chan struct{}),
		workflowID: "canvas-1",
	}

	hub.register <- client

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if hub.WorkflowSubscriberCount("canvas-1") == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if hub.WorkflowSubscriberCount("canvas-1") != 1 {
		t.Fatal("expected subscriber after register")
	}

	hub.BroadcastToWorkflow("canvas-1", []byte("hello"))
	select {
	case msg := <-client.send:
		if string(msg) != "hello" {
			t.Fatalf("got %q, want hello", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("expected broadcast message")
	}

	hub.unregister <- client

	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if hub.WorkflowSubscriberCount("canvas-1") == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected no subscribers after unregister")
}

func TestBroadcastToWorkflow_ConcurrentUnregisterDoesNotPanic(t *testing.T) {
	hub := NewHub()
	hub.Run()

	const n = 32
	clients := make([]*Client, 0, n)
	for i := 0; i < n; i++ {
		client := &Client{
			hub:        hub,
			send:       make(chan []byte, 1),
			Done:       make(chan struct{}),
			workflowID: "canvas-race",
		}
		hub.register <- client
		clients = append(clients, client)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if hub.WorkflowSubscriberCount("canvas-race") == n {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if hub.WorkflowSubscriberCount("canvas-race") != n {
		t.Fatalf("expected %d subscribers, got %d", n, hub.WorkflowSubscriberCount("canvas-race"))
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			hub.BroadcastToWorkflow("canvas-race", []byte("ping"))
		}
	}()
	go func() {
		defer wg.Done()
		for _, client := range clients {
			hub.unregister <- client
		}
	}()
	wg.Wait()
}
