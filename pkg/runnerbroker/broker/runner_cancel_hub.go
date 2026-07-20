package broker

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/superplanehq/superplane/pkg/runnerbroker/wsrunner"
)

// RunnerCancelHub maps a runner WebSocket connection to its active task so POST /cancel
// can push a cancel frame immediately. All writes on the registered conn must use the same
// writeMu as the runner stream goroutine (gorilla/websocket: one writer at a time).
type RunnerCancelHub struct {
	mu       sync.Mutex
	byRunner map[string]*runnerCancelEntry
}

type runnerCancelEntry struct {
	taskID  string
	conn    *websocket.Conn
	writeMu *sync.Mutex
}

// NewRunnerCancelHub returns an empty hub.
func NewRunnerCancelHub() *RunnerCancelHub {
	return &RunnerCancelHub{byRunner: make(map[string]*runnerCancelEntry)}
}

// Register records the active task for runnerID. Last register wins. Unregister must be
// called when the task leaves the in-flight phase (complete, disconnect, or stream error).
func (h *RunnerCancelHub) Register(runnerID, taskID string, conn *websocket.Conn, writeMu *sync.Mutex) (unregister func()) {
	if h == nil || runnerID == "" || taskID == "" || conn == nil || writeMu == nil {
		return func() {}
	}
	h.mu.Lock()
	h.byRunner[runnerID] = &runnerCancelEntry{
		taskID:  taskID,
		conn:    conn,
		writeMu: writeMu,
	}
	h.mu.Unlock()
	return func() { h.unregister(runnerID, taskID) }
}

func (h *RunnerCancelHub) unregister(runnerID, taskID string) {
	if h == nil || runnerID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	ent, ok := h.byRunner[runnerID]
	if !ok || ent.taskID != taskID {
		return
	}
	delete(h.byRunner, runnerID)
}

// PushCancel writes a cancel JSON frame if runnerID has this task registered.
func (h *RunnerCancelHub) PushCancel(runnerID, taskID string) bool {
	if h == nil || runnerID == "" || taskID == "" {
		return false
	}
	h.mu.Lock()
	ent := h.byRunner[runnerID]
	h.mu.Unlock()
	if ent == nil || ent.taskID != taskID || ent.conn == nil || ent.writeMu == nil {
		return false
	}
	ent.writeMu.Lock()
	defer ent.writeMu.Unlock()
	_ = ent.conn.SetWriteDeadline(time.Now().Add(runnerStreamWriteWait))
	err := ent.conn.WriteJSON(wsrunner.Cancel{
		Type:   wsrunner.TypeCancel,
		TaskID: taskID,
	})
	return err == nil
}
