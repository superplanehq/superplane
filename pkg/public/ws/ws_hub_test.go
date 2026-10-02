package ws

import (
"sync"
"testing"
"time"
)

func makeTestClient(h *Hub, workflowID string) *Client {
c := &Client{
hub:        h,
send:       make(chan []byte, 4096),
Done:       make(chan struct{}),
workflowID: workflowID,
}
h.registerClient(c)
return c
}

func makeFullClient(h *Hub, workflowID string) *Client {
c := &Client{
hub:        h,
send:       make(chan []byte, 1),
Done:       make(chan struct{}),
workflowID: workflowID,
}
h.registerClient(c)
c.send <- []byte("fill")
return c
}

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

func TestBroadcastToWorkflow_StalledClientDoesNotDeadlock(t *testing.T) {
h := NewHub()
wfID := "workflow-deadlock-test"

healthy := makeTestClient(h, wfID)
_ = makeFullClient(h, wfID)

done := make(chan struct{})
go func() {
defer close(done)
h.BroadcastToWorkflow(wfID, []byte("hello"))
}()

select {
case <-done:
case <-time.After(2 * time.Second):
t.Fatal("BroadcastToWorkflow deadlocked")
}

select {
case msg := <-healthy.send:
if string(msg) != "hello" {
t.Fatalf("healthy client got %q, want %q", msg, "hello")
}
default:
t.Fatal("healthy client did not receive the broadcast message")
}
}

func TestBroadcastAll_StalledClientDoesNotDeadlock(t *testing.T) {
h := NewHub()

healthy := makeTestClient(h, "wf-all-healthy")
_ = makeFullClient(h, "wf-all-stalled")

done := make(chan struct{})
go func() {
defer close(done)
h.BroadcastAll([]byte("ping"))
}()

select {
case <-done:
case <-time.After(2 * time.Second):
t.Fatal("BroadcastAll deadlocked")
}

select {
case msg := <-healthy.send:
if string(msg) != "ping" {
t.Fatalf("healthy client got %q, want %q", msg, "ping")
}
default:
t.Fatal("healthy client did not receive the broadcast message")
}
}

func TestBroadcastToWorkflow_StalledClientIsEvicted(t *testing.T) {
h := NewHub()
wfID := "workflow-eviction-test"

makeFullClient(h, wfID)
h.BroadcastToWorkflow(wfID, []byte("trigger"))

deadline := time.Now().Add(500 * time.Millisecond)
for time.Now().Before(deadline) {
if h.WorkflowSubscriberCount(wfID) == 0 {
return
}
time.Sleep(2 * time.Millisecond)
}

t.Fatalf("stalled client was not evicted within 500ms; subscriber count = %d",
h.WorkflowSubscriberCount(wfID))
}

func TestBroadcastToWorkflow_MultipleStalledClients(t *testing.T) {
h := NewHub()
wfID := "workflow-multi-stalled"

for i := 0; i < 5; i++ {
makeFullClient(h, wfID)
}
healthy := makeTestClient(h, wfID)

done := make(chan struct{})
go func() {
defer close(done)
h.BroadcastToWorkflow(wfID, []byte("multi"))
}()

select {
case <-done:
case <-time.After(2 * time.Second):
t.Fatal("BroadcastToWorkflow deadlocked with multiple stalled clients")
}

select {
case msg := <-healthy.send:
if string(msg) != "multi" {
t.Fatalf("got %q, want %q", msg, "multi")
}
default:
t.Fatal("healthy client did not receive message")
}
}
