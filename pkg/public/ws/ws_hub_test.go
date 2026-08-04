package ws

import (
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockConn is a mock websocket connection for testing
type MockConn struct {
	mu         sync.Mutex
	closed     bool
	closeCount int32
}

func (m *MockConn) ReadMessage() (int, []byte, error) {
	select {}
}

func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadLimit(limit int64) {}

func (m *MockConn) SetPongHandler(h func(string) error) {}

func (m *MockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		atomic.AddInt32(&m.closeCount, 1)
	}
	return nil
}

func (m *MockConn) NextWriter(messageType int) (io.WriteCloser, error) {
	return &mockWriter{}, nil
}

type mockWriter struct {
	closed bool
}

func (w *mockWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *mockWriter) Close() error {
	w.closed = true
	return nil
}

// TestBroadcastToWorkflowDoesNotDeadlockOnFullClientBuffer verifies that
// broadcasting to a workflow with a stalled client doesn't cause deadlock.
func TestBroadcastToWorkflowDoesNotDeadlockOnFullClientBuffer(t *testing.T) {
	hub := NewHub()
	hub.Run()
	defer func() {
		// Give goroutines time to finish
		time.Sleep(100 * time.Millisecond)
	}()

	// Create a client with a full send buffer
	conn := &MockConn{}
	client := &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 1),
		Done:       make(chan struct{}),
		workflowID: "test-workflow",
	}

	// Fill the send buffer so that the next send will block
	client.send <- []byte("fill")

	// Register the client
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Let registration complete

	// Broadcast to the workflow - this should NOT deadlock
	// Use a timeout to fail if deadlock occurs
	done := make(chan bool)
	go func() {
		hub.BroadcastToWorkflow("test-workflow", []byte("message"))
		done <- true
	}()

	select {
	case <-done:
		// Success - broadcast completed without deadlock
		t.Log("✓ BroadcastToWorkflow completed successfully without deadlock")
	case <-time.After(5 * time.Second):
		t.Fatal("✗ BroadcastToWorkflow deadlocked (timeout after 5s)")
	}
}

// TestBroadcastAllDoesNotDeadlockOnFullClientBuffer verifies that
// broadcasting to all clients with a stalled client doesn't cause deadlock.
func TestBroadcastAllDoesNotDeadlockOnFullClientBuffer(t *testing.T) {
	hub := NewHub()
	hub.Run()
	defer func() {
		time.Sleep(100 * time.Millisecond)
	}()

	// Create a client with a full send buffer
	conn := &MockConn{}
	client := &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 1),
		Done:       make(chan struct{}),
		workflowID: "test-workflow",
	}

	// Fill the send buffer
	client.send <- []byte("fill")

	// Register the client
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Let registration complete

	// Broadcast to all - this should NOT deadlock
	done := make(chan bool)
	go func() {
		hub.BroadcastAll([]byte("message"))
		done <- true
	}()

	select {
	case <-done:
		t.Log("✓ BroadcastAll completed successfully without deadlock")
	case <-time.After(5 * time.Second):
		t.Fatal("✗ BroadcastAll deadlocked (timeout after 5s)")
	}
}

// TestBroadcastToWorkflowDropsStalledClientAndKeepsHealthyOnes verifies that
// stalled clients are unregistered while healthy clients keep receiving messages.
func TestBroadcastToWorkflowDropsStalledClientAndKeepsHealthyOnes(t *testing.T) {
	hub := NewHub()
	hub.Run()
	defer func() {
		time.Sleep(100 * time.Millisecond)
	}()

	// Create a stalled client (full buffer)
	stalledConn := &MockConn{}
	stalledClient := &Client{
		hub:        hub,
		conn:       stalledConn,
		send:       make(chan []byte, 1),
		Done:       make(chan struct{}),
		workflowID: "test-workflow",
	}
	stalledClient.send <- []byte("fill") // Fill buffer

	// Create a healthy client (has room to receive)
	healthyConn := &MockConn{}
	healthyClient := &Client{
		hub:        hub,
		conn:       healthyConn,
		send:       make(chan []byte, 100), // Large buffer
		Done:       make(chan struct{}),
		workflowID: "test-workflow",
	}

	// Register both
	hub.register <- stalledClient
	hub.register <- healthyClient
	time.Sleep(50 * time.Millisecond)

	// Broadcast - should drop stalled client but keep healthy one
	hub.BroadcastToWorkflow("test-workflow", []byte("message"))

	// Give time for async unregistration
	time.Sleep(50 * time.Millisecond)

	// Verify stalled client was unregistered
	hub.mutex.RLock()
	stalledStillRegistered := hub.clients[stalledClient]
	healthyStillRegistered := hub.clients[healthyClient]
	workflowCount := len(hub.workflowSubscriptions["test-workflow"])
	hub.mutex.RUnlock()

	if stalledStillRegistered {
		t.Error("✗ Stalled client should have been unregistered")
	} else {
		t.Log("✓ Stalled client was unregistered")
	}

	if !healthyStillRegistered {
		t.Error("✗ Healthy client should still be registered")
	} else {
		t.Log("✓ Healthy client remained registered")
	}

	// Verify message was received by healthy client
	select {
	case msg := <-healthyClient.send:
		if len(msg) == 0 {
			t.Error("✗ Healthy client received empty message")
		} else {
			t.Log("✓ Healthy client received message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("✗ Healthy client did not receive message")
	}

	// Verify workflow has only 1 client now (stalled dropped)
	if workflowCount != 1 {
		t.Errorf("✗ Workflow should have 1 client, has %d", workflowCount)
	} else {
		t.Log("✓ Workflow has correct client count")
	}
}

// TestConcurrentBroadcastsDropSameStalledClientSafely verifies that
// multiple broadcasts racing to drop the same stalled client is safe.
func TestConcurrentBroadcastsDropSameStalledClientSafely(t *testing.T) {
	hub := NewHub()
	hub.Run()
	defer func() {
		time.Sleep(100 * time.Millisecond)
	}()

	// Create a stalled client
	conn := &MockConn{}
	client := &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 1),
		Done:       make(chan struct{}),
		workflowID: "test-workflow",
	}
	client.send <- []byte("fill")

	// Register the client
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	// Launch 8 concurrent broadcasters - all will try to drop the same client
	numBroadcasters := 8
	var wg sync.WaitGroup
	panics := make(chan interface{}, numBroadcasters)

	for i := 0; i < numBroadcasters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panics <- r
				}
			}()
			hub.BroadcastToWorkflow("test-workflow", []byte("message"))
		}()
	}

	// Wait for all to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("✓ All concurrent broadcasts completed")
	case <-time.After(5 * time.Second):
		t.Fatal("✗ Concurrent broadcasts deadlocked (timeout)")
	}

	// Check for panics
	close(panics)
	panicCount := 0
	for p := range panics {
		panicCount++
		t.Logf("✗ Panic in broadcaster: %v", p)
	}

	if panicCount == 0 {
		t.Log("✓ No panics from concurrent access")
	} else {
		t.Fatalf("✗ %d panics occurred", panicCount)
	}

	// Verify client was only unregistered once
	hub.mutex.RLock()
	isRegistered := hub.clients[client]
	hub.mutex.RUnlock()

	if isRegistered {
		t.Error("✗ Client should have been unregistered by at least one broadcaster")
	} else {
		t.Log("✓ Client was safely unregistered")
	}
}

// TestWorkflowSubscriberCountIsSafe verifies thread-safe subscriber counting
func TestWorkflowSubscriberCountIsSafe(t *testing.T) {
	hub := NewHub()
	hub.Run()

	// Create multiple clients for same workflow
	clients := make([]*Client, 10)
	for i := 0; i < 10; i++ {
		clients[i] = &Client{
			hub:        hub,
			conn:       &MockConn{},
			send:       make(chan []byte, 100),
			Done:       make(chan struct{}),
			workflowID: "test-workflow",
		}
		hub.register <- clients[i]
	}

	time.Sleep(100 * time.Millisecond)

	// Count should be 10
	count := hub.WorkflowSubscriberCount("test-workflow")
	if count != 10 {
		t.Errorf("✗ Expected 10 subscribers, got %d", count)
	} else {
		t.Logf("✓ Correct subscriber count: %d", count)
	}

	// Unregister one
	hub.unregister <- clients[0]
	time.Sleep(50 * time.Millisecond)

	// Count should be 9
	count = hub.WorkflowSubscriberCount("test-workflow")
	if count != 9 {
		t.Errorf("✗ Expected 9 subscribers after unregister, got %d", count)
	} else {
		t.Logf("✓ Correct subscriber count after unregister: %d", count)
	}
}

// BenchmarkBroadcastAll measures broadcast performance
func BenchmarkBroadcastAll(b *testing.B) {
	hub := NewHub()
	hub.Run()

	// Create 100 clients
	for i := 0; i < 100; i++ {
		client := &Client{
			hub:        hub,
			conn:       &MockConn{},
			send:       make(chan []byte, 100),
			Done:       make(chan struct{}),
			workflowID: "bench-workflow",
		}
		hub.register <- client
	}

	time.Sleep(100 * time.Millisecond)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hub.BroadcastAll([]byte("benchmark message"))
	}
}

// BenchmarkBroadcastToWorkflow measures targeted broadcast performance
func BenchmarkBroadcastToWorkflow(b *testing.B) {
	hub := NewHub()
	hub.Run()

	// Create 100 clients for same workflow
	for i := 0; i < 100; i++ {
		client := &Client{
			hub:        hub,
			conn:       &MockConn{},
			send:       make(chan []byte, 100),
			Done:       make(chan struct{}),
			workflowID: "bench-workflow",
		}
		hub.register <- client
	}

	time.Sleep(100 * time.Millisecond)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hub.BroadcastToWorkflow("bench-workflow", []byte("benchmark message"))
	}
}