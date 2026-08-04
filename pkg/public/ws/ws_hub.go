package ws

import (
"context"
"io"
"net"
"sync"
"time"

"github.com/gorilla/websocket"
log "github.com/sirupsen/logrus"
"github.com/superplanehq/superplane/pkg/telemetry"
)

const (
writeWait  = 10 * time.Second
pongWait   = 20 * time.Second
pingPeriod = 10 * time.Second
)

type Client struct {
hub        *Hub
conn       *websocket.Conn
send       chan []byte
Done       chan struct{}
workflowID string
}

type Hub struct {
clients               map[*Client]bool
workflowSubscriptions map[string]map[*Client]bool
register              chan *Client
unregister            chan *Client
mutex                 sync.RWMutex
}

func NewHub() *Hub {
return &Hub{
clients:               make(map[*Client]bool),
workflowSubscriptions: make(map[string]map[*Client]bool),
register:              make(chan *Client),
unregister:            make(chan *Client),
mutex:                 sync.RWMutex{},
}
}

func (h *Hub) Run() {
go func() {
for {
select {
case client := <-h.register:
h.registerClient(client)
case client := <-h.unregister:
h.unregisterClient(client)
}
}
}()
}

func (h *Hub) registerClient(client *Client) {
h.mutex.Lock()
defer h.mutex.Unlock()

h.clients[client] = true

if _, ok := h.workflowSubscriptions[client.workflowID]; !ok {
h.workflowSubscriptions[client.workflowID] = make(map[*Client]bool)
}

h.workflowSubscriptions[client.workflowID][client] = true

log.Debugf("Client subscribed to workflow: %s", client.workflowID)
log.Debugf("New client registered %v, total clients: %d", client, len(h.clients))
telemetry.RecordWebSocketConnectionOpened(context.Background(), KindFromTopic(client.workflowID))
}

// unregisterClient removes a client from the hub.
// Callers must NOT hold h.mutex when calling this.
func (h *Hub) unregisterClient(client *Client) {
h.mutex.Lock()
defer h.mutex.Unlock()
h.unregisterClientLocked(client)
}

// unregisterClientLocked removes a client. Caller must hold h.mutex write lock.
func (h *Hub) unregisterClientLocked(client *Client) {
if _, ok := h.clients[client]; !ok {
return
}

delete(h.clients, client)
close(client.send)

if client.workflowID != "" {
if clients, ok := h.workflowSubscriptions[client.workflowID]; ok {
delete(clients, client)
if len(clients) == 0 {
delete(h.workflowSubscriptions, client.workflowID)
}
}
}

log.Debugf("Client unregistered, remaining clients: %d", len(h.clients))
telemetry.RecordWebSocketConnectionClosed(context.Background(), KindFromTopic(client.workflowID))
}

// BroadcastAll sends a message to all connected clients.
//
// Stalled clients (full send buffer) are collected while holding the read
// lock and then evicted after releasing it, so we never call unregisterClient
// — which acquires a write lock — while the read lock is still held.
// sync.RWMutex is not reentrant; acquiring a write lock while holding a read
// lock on the same mutex deadlocks the goroutine.
func (h *Hub) BroadcastAll(message []byte) {
var stalled []*Client

h.mutex.RLock()
for client := range h.clients {
select {
case client.send <- message:
default:
stalled = append(stalled, client)
}
}
h.mutex.RUnlock()

for _, client := range stalled {
log.Warnf("WebSocket client send buffer full, evicting client for workflow %q", client.workflowID)
h.unregisterClient(client)
}
}

// BroadcastToWorkflow sends a message to all clients subscribed to workflowID.
//
// Stalled clients are collected under the read lock and evicted after
// releasing it, so the write lock is never acquired while the read lock
// is still held (sync.RWMutex is not reentrant).
func (h *Hub) BroadcastToWorkflow(workflowID string, message []byte) {
kind := KindFromTopic(workflowID)
var stalled []*Client
recipients := 0

h.mutex.RLock()
clients := h.workflowSubscriptions[workflowID]
if len(clients) == 0 {
h.mutex.RUnlock()
telemetry.RecordWebSocketBroadcast(context.Background(), kind, 0)
return
}

for client := range clients {
select {
case client.send <- message:
recipients++
default:
stalled = append(stalled, client)
}
}
h.mutex.RUnlock()

telemetry.RecordWebSocketBroadcast(context.Background(), kind, recipients)

for _, client := range stalled {
log.Warnf("WebSocket client send buffer full, evicting client for workflow %q", client.workflowID)
h.unregisterClient(client)
}
}

func (h *Hub) WorkflowSubscriberCount(workflowID string) int {
h.mutex.RLock()
defer h.mutex.RUnlock()
return len(h.workflowSubscriptions[workflowID])
}

func (h *Hub) NewClient(conn *websocket.Conn, workflowID string) *Client {
client := &Client{
hub:        h,
conn:       conn,
send:       make(chan []byte, 4096),
Done:       make(chan struct{}),
workflowID: workflowID,
}

h.register <- client
go client.writePump()
go client.readPump()

return client
}

func (c *Client) writePump() {
ticker := time.NewTicker(pingPeriod)
defer func() {
ticker.Stop()
c.conn.Close()
}()

for {
select {
case message, ok := <-c.send:
c.conn.SetWriteDeadline(time.Now().Add(writeWait))
if !ok {
c.conn.WriteMessage(websocket.CloseMessage, []byte{})
return
}

w, err := c.conn.NextWriter(websocket.TextMessage)
if err != nil {
return
}

w.Write(message)

if err := w.Close(); err != nil {
return
}

case <-ticker.C:
c.conn.SetWriteDeadline(time.Now().Add(writeWait))
if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
return
}
}
}
}

func (c *Client) readPump() {
defer func() {
c.hub.unregister <- c
close(c.Done)
c.conn.Close()
}()

c.conn.SetReadLimit(1024 * 1024)
c.conn.SetReadDeadline(time.Now().Add(pongWait))
c.conn.SetPongHandler(func(string) error {
c.conn.SetReadDeadline(time.Now().Add(pongWait))
return nil
})

for {
_, message, err := c.conn.ReadMessage()
if err != nil {
if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
log.Errorf("Unexpected WebSocket closure: %v", err)
} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
log.Info("WebSocket closed normally")
} else if err == io.EOF {
log.Info("WebSocket connection EOF")
} else if ne, ok := err.(*net.OpError); ok {
log.Warnf("WebSocket network error: %v", ne)
} else {
log.Warnf("WebSocket read error: %v", err)
}
break
}

c.handleMessage(message)
}
}

// handleMessage processes incoming messages from clients.
// Logged at Debug level to avoid a log-flood vector from a single connection.
func (c *Client) handleMessage(message []byte) {
log.Debugf("Received message: %s", string(message))
}
