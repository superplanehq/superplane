package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
	appjwt "github.com/superplanehq/superplane/pkg/jwt"
)

var ErrNoWorkersAvailable = errors.New("no workers available")

type DispatchRequest struct {
	OrganizationID string
	WorkerPoolID   string
	ExtensionID    string
	VersionID      string
	Digest         string
	Invocation     json.RawMessage
}

type DispatchResult struct {
	Output json.RawMessage
}

const bundleTokenSubject = "extension-bundle"

type WorkerInfo struct {
	WorkerID       string
	OrganizationID string
	WorkerPoolID   string
	Busy           bool
	ConnectedAt    time.Time
	LastSeenAt     time.Time
}

type Hub struct {
	storage  *extensions.Storage
	signer   *appjwt.Signer
	upgrader websocket.Upgrader

	mu      sync.Mutex
	workers map[string]*workerSession
	jobs    map[string]*jobState
}

type workerSession struct {
	registration protocol.Registration
	conn         *websocket.Conn
	send         chan any
	done         chan struct{}
	connectedAt  time.Time
	lastSeenAt   time.Time
	busy         bool
	closeOnce    sync.Once
}

type jobState struct {
	id       string
	worker   *workerSession
	resultCh chan dispatchOutcome
}

type dispatchOutcome struct {
	result DispatchResult
	err    error
}

func New(storage *extensions.Storage, signer *appjwt.Signer) *Hub {
	return &Hub{
		storage: storage,
		signer:  signer,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		workers: make(map[string]*workerSession),
		jobs:    make(map[string]*jobState),
	}
}

func (h *Hub) ListWorkers() []WorkerInfo {
	h.mu.Lock()
	defer h.mu.Unlock()

	workers := make([]WorkerInfo, 0, len(h.workers))
	for _, worker := range h.workers {
		workers = append(workers, WorkerInfo{
			WorkerID:       worker.registration.WorkerID,
			OrganizationID: worker.registration.OrganizationID,
			WorkerPoolID:   worker.registration.WorkerPoolID,
			Busy:           worker.busy,
			ConnectedAt:    worker.connectedAt,
			LastSeenAt:     worker.lastSeenAt,
		})
	}

	return workers
}

func (h *Hub) Dispatch(ctx context.Context, request DispatchRequest) (DispatchResult, error) {
	h.mu.Lock()

	worker := h.findAvailableWorkerLocked(request.OrganizationID, request.WorkerPoolID)
	if worker == nil {
		h.mu.Unlock()
		return DispatchResult{}, ErrNoWorkersAvailable
	}

	jobID := uuid.NewString()
	state := &jobState{
		id:       jobID,
		worker:   worker,
		resultCh: make(chan dispatchOutcome, 1),
	}

	worker.busy = true
	h.jobs[jobID] = state
	h.mu.Unlock()

	bundleToken, err := h.generateBundleToken(request)
	if err != nil {
		h.failJob(jobID, fmt.Errorf("generate bundle token: %w", err))
		return DispatchResult{}, err
	}

	message := protocol.JobAssignMessage{
		Type:        protocol.MessageTypeJobAssign,
		JobID:       jobID,
		ExtensionID: request.ExtensionID,
		VersionID:   request.VersionID,
		Digest:      request.Digest,
		BundleToken: bundleToken,
		Invocation:  request.Invocation,
	}

	if err := h.queueMessage(worker, message); err != nil {
		h.failJob(jobID, fmt.Errorf("send job assignment: %w", err))
		return DispatchResult{}, err
	}

	select {
	case <-ctx.Done():
		h.cancelJob(jobID)
		return DispatchResult{}, ctx.Err()
	case outcome := <-state.resultCh:
		return outcome.result, outcome.err
	}
}

func (h *Hub) registerWorker(registration protocol.Registration, conn *websocket.Conn) *workerSession {
	session := &workerSession{
		registration: registration,
		conn:         conn,
		send:         make(chan any, 16),
		done:         make(chan struct{}),
		connectedAt:  time.Now().UTC(),
		lastSeenAt:   time.Now().UTC(),
	}

	key := workerKey(registration)

	h.mu.Lock()
	if existing, ok := h.workers[key]; ok {
		go h.unregisterWorker(existing, fmt.Errorf("worker re-registered"))
	}
	h.workers[key] = session
	h.mu.Unlock()

	go h.writeLoop(session)
	go h.readLoop(session)

	return session
}

func (h *Hub) queueMessage(worker *workerSession, message any) error {
	select {
	case <-worker.done:
		return fmt.Errorf("worker %s is disconnected", worker.registration.WorkerID)
	case worker.send <- message:
		return nil
	default:
		return fmt.Errorf("worker %s send queue is full", worker.registration.WorkerID)
	}
}

func (h *Hub) readLoop(worker *workerSession) {
	defer h.unregisterWorker(worker, fmt.Errorf("worker connection closed"))

	for {
		_, payload, err := worker.conn.ReadMessage()
		if err != nil {
			return
		}

		if err := h.handleWorkerMessage(worker, payload); err != nil {
			return
		}
	}
}

func (h *Hub) writeLoop(worker *workerSession) {
	ticker := time.NewTicker(15 * time.Second)
	defer func() {
		ticker.Stop()
		h.unregisterWorker(worker, fmt.Errorf("worker writer stopped"))
	}()

	for {
		select {
		case <-worker.done:
			return
		case message, ok := <-worker.send:
			if !ok {
				return
			}

			if err := worker.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			if err := worker.conn.WriteJSON(protocol.NewPing()); err != nil {
				return
			}
		}
	}
}

func (h *Hub) handleWorkerMessage(worker *workerSession, payload []byte) error {
	var envelope protocol.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode worker message: %w", err)
	}

	h.mu.Lock()
	worker.lastSeenAt = time.Now().UTC()
	h.mu.Unlock()

	switch envelope.Type {
	case protocol.MessageTypeJobComplete:
		var message protocol.JobCompleteMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			return err
		}

		if message.Success {
			h.completeJob(message.JobID, DispatchResult{Output: message.Output})
			return nil
		}

		if message.Error == nil {
			h.failJob(message.JobID, fmt.Errorf("job failed without error details"))
			return nil
		}

		h.failJob(message.JobID, fmt.Errorf("%s: %s", message.Error.Code, message.Error.Message))
		return nil
	case protocol.MessageTypePong:
		return nil
	default:
		return nil
	}
}

func (h *Hub) completeJob(jobID string, result DispatchResult) {
	h.finishJob(jobID, dispatchOutcome{result: result})
}

func (h *Hub) failJob(jobID string, err error) {
	h.finishJob(jobID, dispatchOutcome{err: err})
}

func (h *Hub) finishJob(jobID string, outcome dispatchOutcome) {
	h.mu.Lock()
	state, ok := h.jobs[jobID]
	if !ok {
		h.mu.Unlock()
		return
	}

	delete(h.jobs, jobID)
	if state.worker != nil {
		state.worker.busy = false
	}
	h.mu.Unlock()
	state.resultCh <- outcome
}

func (h *Hub) cancelJob(jobID string) {
	h.mu.Lock()
	state, ok := h.jobs[jobID]
	if !ok {
		h.mu.Unlock()
		return
	}

	delete(h.jobs, jobID)
	if state.worker != nil {
		state.worker.busy = false
		_ = h.queueMessage(state.worker, protocol.JobCancelMessage{
			Type:  protocol.MessageTypeJobCancel,
			JobID: jobID,
		})
	}
	h.mu.Unlock()
}

func (h *Hub) unregisterWorker(worker *workerSession, reason error) {
	worker.closeOnce.Do(func() {
		key := workerKey(worker.registration)

		h.mu.Lock()
		if current, ok := h.workers[key]; ok && current == worker {
			delete(h.workers, key)
		}

		failedJobs := []*jobState{}
		for jobID, state := range h.jobs {
			if state.worker != worker {
				continue
			}

			delete(h.jobs, jobID)
			failedJobs = append(failedJobs, state)
		}
		h.mu.Unlock()

		close(worker.done)
		if worker.conn != nil {
			_ = worker.conn.Close()
		}

		for _, state := range failedJobs {
			state.resultCh <- dispatchOutcome{err: reason}
		}
	})
}

func (h *Hub) findAvailableWorkerLocked(organizationID string, workerPoolID string) *workerSession {
	for _, worker := range h.workers {
		if worker.registration.OrganizationID != organizationID {
			continue
		}
		if worker.registration.WorkerPoolID != workerPoolID {
			continue
		}
		if worker.busy {
			continue
		}

		return worker
	}

	return nil
}

func workerKey(registration protocol.Registration) string {
	return fmt.Sprintf("%s:%s:%s", registration.OrganizationID, registration.WorkerPoolID, registration.WorkerID)
}

func (h *Hub) generateBundleToken(request DispatchRequest) (string, error) {
	if h.signer == nil {
		return "", fmt.Errorf("hub signer is not configured")
	}

	return h.signer.GenerateWithClaims(bundleTokenSubject, 15*time.Minute, map[string]any{
		"organizationId": request.OrganizationID,
		"extensionId":    request.ExtensionID,
		"versionId":      request.VersionID,
	})
}
