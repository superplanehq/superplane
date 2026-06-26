package broker

import (
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/superplane/runner/shared/api"
)

const runnerDrainClaiming = "__claiming__"

type runnerDrainEntry struct {
	fleetID      string
	conn         *websocket.Conn
	activeTaskID string
	draining     bool
}

// RunnerDrainHub tracks runner streams selected for EC2 termination.
// Draining a runner is sticky for its EC2 instance id, so a reconnecting runner
// cannot receive new work after fleet-manager has decided to terminate it.
type RunnerDrainHub struct {
	mu       sync.Mutex
	byRunner map[string]*runnerDrainEntry
}

func NewRunnerDrainHub() *RunnerDrainHub {
	return &RunnerDrainHub{byRunner: make(map[string]*runnerDrainEntry)}
}

func (h *RunnerDrainHub) Register(runnerID, fleetID string, conn *websocket.Conn) func() {
	runnerID = strings.TrimSpace(runnerID)
	fleetID = strings.TrimSpace(fleetID)
	if h == nil || runnerID == "" || fleetID == "" {
		return func() {}
	}

	h.mu.Lock()
	entry := h.entryLocked(runnerID)
	entry.fleetID = fleetID
	entry.conn = conn
	h.mu.Unlock()

	return func() {
		h.mu.Lock()
		if cur := h.byRunner[runnerID]; cur == entry {
			cur.conn = nil
			cur.activeTaskID = ""
		}
		h.mu.Unlock()
	}
}

// TryStartClaim reserves the runner for ClaimTask unless fleet-manager has already
// started draining it. FinishClaim must be called after the claim attempt.
func (h *RunnerDrainHub) TryStartClaim(runnerID string) bool {
	runnerID = strings.TrimSpace(runnerID)
	if h == nil || runnerID == "" {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := h.byRunner[runnerID]
	if entry != nil && entry.draining {
		return false
	}
	if entry == nil {
		entry = h.entryLocked(runnerID)
	}
	entry.activeTaskID = runnerDrainClaiming
	return true
}

func (h *RunnerDrainHub) FinishClaim(runnerID, taskID string) {
	runnerID = strings.TrimSpace(runnerID)
	taskID = strings.TrimSpace(taskID)
	if h == nil || runnerID == "" {
		return
	}
	h.mu.Lock()
	if entry := h.byRunner[runnerID]; entry != nil {
		entry.activeTaskID = taskID
	}
	h.mu.Unlock()
}

func (h *RunnerDrainHub) CompleteTask(runnerID, taskID string) {
	runnerID = strings.TrimSpace(runnerID)
	taskID = strings.TrimSpace(taskID)
	if h == nil || runnerID == "" {
		return
	}
	h.mu.Lock()
	if entry := h.byRunner[runnerID]; entry != nil {
		if taskID == "" || entry.activeTaskID == taskID || entry.activeTaskID == runnerDrainClaiming {
			entry.activeTaskID = ""
		}
	}
	h.mu.Unlock()
}

func (h *RunnerDrainHub) IsDraining(runnerID string) bool {
	runnerID = strings.TrimSpace(runnerID)
	if h == nil || runnerID == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	entry := h.byRunner[runnerID]
	return entry != nil && entry.draining
}

func (h *RunnerDrainHub) Drain(fleetID string, runnerIDs []string) []api.DrainRunnerStatus {
	fleetID = strings.TrimSpace(fleetID)
	ids := compactRunnerIDs(runnerIDs)
	if h == nil || fleetID == "" {
		return drainStatuses(ids, api.DrainRunnerStateBusy)
	}

	closeIdle := make([]*websocket.Conn, 0, len(ids))
	out := make([]api.DrainRunnerStatus, 0, len(ids))
	h.mu.Lock()
	for _, runnerID := range ids {
		entry := h.entryLocked(runnerID)
		if entry.fleetID == "" {
			entry.fleetID = fleetID
		}
		status := api.DrainRunnerStatus{RunnerID: runnerID}
		if entry.fleetID != fleetID {
			status.State = api.DrainRunnerStateBusy
			out = append(out, status)
			continue
		}

		entry.draining = true
		if entry.activeTaskID != "" {
			status.State = api.DrainRunnerStateBusy
			if entry.activeTaskID != runnerDrainClaiming {
				status.ActiveTaskID = entry.activeTaskID
			}
			out = append(out, status)
			continue
		}

		status.State = api.DrainRunnerStateDrained
		out = append(out, status)
		if entry.conn != nil {
			closeIdle = append(closeIdle, entry.conn)
		}
	}
	h.mu.Unlock()

	for _, conn := range closeIdle {
		_ = conn.Close()
	}
	return out
}

func (h *RunnerDrainHub) entryLocked(runnerID string) *runnerDrainEntry {
	entry := h.byRunner[runnerID]
	if entry == nil {
		entry = &runnerDrainEntry{}
		h.byRunner[runnerID] = entry
	}
	return entry
}

func drainStatuses(ids []string, state string) []api.DrainRunnerStatus {
	out := make([]api.DrainRunnerStatus, 0, len(ids))
	for _, id := range ids {
		out = append(out, api.DrainRunnerStatus{RunnerID: id, State: state})
	}
	return out
}

func compactRunnerIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
