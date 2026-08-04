package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// createWebSocketPairViaServer creates a proper websocket pair via an httptest server
func createWebSocketPairViaServer(t *testing.T) (serverConn *websocket.Conn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()

	var serverWs *websocket.Conn
	var mu sync.Mutex
	connected := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		serverWs = ws
		mu.Unlock()
		close(connected)
	}))

	// Connect client
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	clientWs, _, err := dialer.Dial(url, nil)
	require.NoError(t, err)

	// Wait for server to accept
	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not accept connection in time")
	}

	return serverWs, clientWs, func() {
		serverWs.Close()
		clientWs.Close()
		server.Close()
	}
}

// TestBroadcastAll_DeadlockFix verifies that broadcasting to a stalled client
// (full send buffer) does not deadlock the hub
func TestBroadcastAll_DeadlockFix(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn, cleanup := createWebSocketPairViaServer(t)
	defer cleanup()

	client := &Client{
		hub:        hub,
		conn:       serverConn,
		send:       make(chan []byte, 256), // Use small buffer for faster fill
		Done:       make(chan struct{}),
		workflowID: "workflow-1",
	}
	_ = clientConn // Client-side conn kept open to avoid EOF

	// Register client directly
	hub.registerClient(client)
	defer func() {
		hub.unregisterClient(client)
		close(client.Done)
	}()

	// Fill the send buffer completely
	for i := 0; i < 256; i++ {
		select {
		case client.send <- []byte("fill"):
		default:
			t.Fatal("Could not fill send buffer")
		}
	}

	// Now broadcast - this should NOT deadlock even though the client's buffer is full
	done := make(chan struct{})
	go func() {
		hub.BroadcastAll([]byte("test message"))
		close(done)
	}()

	select {
	case <-done:
		// Success - broadcast completed without deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("BroadcastAll deadlocked - hub is stuck")
	}

	// Give the hub a moment to process unregistrations
	time.Sleep(50 * time.Millisecond)

	// Verify the stalled client was evicted
	hub.mutex.RLock()
	_, exists := hub.clients[client]
	hub.mutex.RUnlock()
	assert.False(t, exists, "Stalled client should have been evicted")
}

// TestBroadcastToWorkflow_DeadlockFix verifies that broadcasting to a specific workflow
// with stalled clients does not deadlock the hub
func TestBroadcastToWorkflow_DeadlockFix(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn, cleanup := createWebSocketPairViaServer(t)
	defer cleanup()

	client := &Client{
		hub:        hub,
		conn:       serverConn,
		send:       make(chan []byte, 256),
		Done:       make(chan struct{}),
		workflowID: "workflow-1",
	}
	_ = clientConn

	hub.registerClient(client)
	defer func() {
		hub.unregisterClient(client)
		close(client.Done)
	}()

	// Fill the send buffer
	for i := 0; i < 256; i++ {
		select {
		case client.send <- []byte("fill"):
		default:
			t.Fatal("Could not fill send buffer")
		}
	}

	// Broadcast to the workflow - should NOT deadlock
	done := make(chan struct{})
	go func() {
		hub.BroadcastToWorkflow("workflow-1", []byte("test message"))
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("BroadcastToWorkflow deadlocked - hub is stuck")
	}

	time.Sleep(50 * time.Millisecond)

	hub.mutex.RLock()
	_, exists := hub.clients[client]
	hub.mutex.RUnlock()
	assert.False(t, exists, "Stalled client should have been evicted")
}

// TestBroadcast_MultipleStalledClients verifies that multiple stalled clients
// are all evicted without deadlock
func TestBroadcast_MultipleStalledClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var stalledClients []*Client
	var cleanups []func()

	// Create 5 stalled clients
	for i := 0; i < 5; i++ {
		serverConn, clientConn, cleanup := createWebSocketPairViaServer(t)
		cleanups = append(cleanups, cleanup)
		_ = clientConn

		client := &Client{
			hub:        hub,
			conn:       serverConn,
			send:       make(chan []byte, 256),
			Done:       make(chan struct{}),
			workflowID: "workflow-1",
		}
		hub.registerClient(client)

		// Fill the send buffer
		for j := 0; j < 256; j++ {
			select {
			case client.send <- []byte("fill"):
			default:
				t.Fatalf("Could not fill send buffer for client %d", i)
			}
		}
		stalledClients = append(stalledClients, client)
	}

	// Create 2 healthy clients
	var healthyClients []*Client
	for i := 0; i < 2; i++ {
		serverConn, clientConn, cleanup := createWebSocketPairViaServer(t)
		cleanups = append(cleanups, cleanup)
		_ = clientConn

		client := &Client{
			hub:        hub,
			conn:       serverConn,
			send:       make(chan []byte, 256),
			Done:       make(chan struct{}),
			workflowID: "workflow-1",
		}
		hub.registerClient(client)
		healthyClients = append(healthyClients, client)
	}

	// Broadcast should NOT deadlock and should evict only stalled clients
	done := make(chan struct{})
	go func() {
		hub.BroadcastToWorkflow("workflow-1", []byte("test message"))
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcast deadlocked with multiple stalled clients")
	}

	time.Sleep(50 * time.Millisecond)

	hub.mutex.RLock()
	for i, client := range stalledClients {
		_, exists := hub.clients[client]
		assert.False(t, exists, "Stalled client %d should have been evicted", i)
	}
	for i, client := range healthyClients {
		_, exists := hub.clients[client]
		assert.True(t, exists, "Healthy client %d should still be registered", i)
	}
	hub.mutex.RUnlock()

	for _, cleanup := range cleanups {
		cleanup()
	}
}

// TestBroadcast_ConcurrentSafety verifies that concurrent broadcasts
// with stalled clients do not cause race conditions
func TestBroadcast_ConcurrentSafety(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var cleanups []func()

	// Create 10 clients with mixed states (every other one stalled)
	for i := 0; i < 10; i++ {
		serverConn, clientConn, cleanup := createWebSocketPairViaServer(t)
		cleanups = append(cleanups, cleanup)
		_ = clientConn

		client := &Client{
			hub:        hub,
			conn:       serverConn,
			send:       make(chan []byte, 256),
			Done:       make(chan struct{}),
			workflowID: "workflow-1",
		}
		hub.registerClient(client)

		// Make every other client stalled
		if i%2 == 0 {
			for j := 0; j < 256; j++ {
				select {
				case client.send <- []byte("fill"):
				default:
					break
				}
			}
		}
	}

	// Run concurrent broadcasts
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.BroadcastAll([]byte("concurrent test"))
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent broadcasts deadlocked")
	}

	for _, cleanup := range cleanups {
		cleanup()
	}
}

// TestBroadcast_NoStalledClients verifies normal operation when no clients are stalled
func TestBroadcast_NoStalledClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn1, clientConn1, cleanup1 := createWebSocketPairViaServer(t)
	defer cleanup1()
	_ = clientConn1

	client1 := &Client{
		hub:        hub,
		conn:       serverConn1,
		send:       make(chan []byte, 256),
		Done:       make(chan struct{}),
		workflowID: "workflow-1",
	}
	hub.registerClient(client1)

	serverConn2, clientConn2, cleanup2 := createWebSocketPairViaServer(t)
	defer cleanup2()
	_ = clientConn2

	client2 := &Client{
		hub:        hub,
		conn:       serverConn2,
		send:       make(chan []byte, 256),
		Done:       make(chan struct{}),
		workflowID: "workflow-1",
	}
	hub.registerClient(client2)

	// Broadcast should succeed normally
	done := make(chan struct{})
	go func() {
		hub.BroadcastToWorkflow("workflow-1", []byte("normal message"))
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Normal broadcast timed out")
	}

	// Verify both clients received the message
	time.Sleep(50 * time.Millisecond)
	assert.Len(t, client1.send, 1, "Client 1 should have received message")
	assert.Len(t, client2.send, 1, "Client 2 should have received message")

	// Verify both clients are still registered
	hub.mutex.RLock()
	_, exists1 := hub.clients[client1]
	_, exists2 := hub.clients[client2]
	hub.mutex.RUnlock()
	assert.True(t, exists1, "Client 1 should still be registered")
	assert.True(t, exists2, "Client 2 should still be registered")
}
