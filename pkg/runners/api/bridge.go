package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
)

// FleetSync is POST /runner-fleets/sync (fleet-manager pulls the next job).
func (h *Handler) FleetSync(w http.ResponseWriter, r *http.Request) {
	fleet, ok := h.authenticateRunnerFleet(w, r)
	if !ok {
		return
	}

	task, err := h.store().ClaimNextQueuedJob(fleet.ID)
	if err != nil {
		log.Errorf("runner fleet sync: claim job: %v", err)
		http.Error(w, "could not claim job", http.StatusInternalServerError)
		return
	}

	resp := runnermodels.FleetSyncResponse{Continue: task != nil}
	if task != nil {
		resp.Job = &runnermodels.FleetBridgeJob{
			ID:   task.ID.String(),
			Spec: task.Spec.Data(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// FleetTaskComplete is POST /runner-fleets/tasks/{taskId}/complete.
func (h *Handler) FleetTaskComplete(w http.ResponseWriter, r *http.Request) {
	fleet, ok := h.authenticateRunnerFleet(w, r)
	if !ok {
		return
	}

	taskIDStr := strings.TrimSpace(mux.Vars(r)["taskId"])
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var req runnermodels.FleetCompleteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.store().FindTask(taskID)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if task.FleetID != fleet.ID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	wasTerminal := task.IsTerminal()
	task, err = h.store().CompleteJob(taskID, req)
	if err != nil {
		log.Errorf("runner fleet complete: %v", err)
		http.Error(w, "could not complete task", http.StatusInternalServerError)
		return
	}

	if wasTerminal {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.finishRunnerTask(task); err != nil {
		log.Errorf("runner fleet complete: finish execution: %v", err)
		http.Error(w, "could not finish execution", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) authenticateRunnerFleet(w http.ResponseWriter, r *http.Request) (*runnermodels.RunnerFleet, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	fleet, err := h.store().FindFleetByAuthToken(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return fleet, true
}
