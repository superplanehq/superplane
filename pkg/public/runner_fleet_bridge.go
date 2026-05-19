package public

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/runners"
)

func (s *Server) handleRunnerFleetSync(w http.ResponseWriter, r *http.Request) {
	fleet, ok := s.authenticateRunnerFleet(w, r)
	if !ok {
		return
	}

	store := runners.NewPostgresStore()
	task, err := store.ClaimNextQueuedJob(fleet.ID)
	if err != nil {
		log.Errorf("runner fleet sync: claim job: %v", err)
		http.Error(w, "could not claim job", http.StatusInternalServerError)
		return
	}

	resp := runners.FleetSyncResponse{Continue: task != nil}
	if task != nil {
		resp.Job = &runners.FleetBridgeJob{
			ID:   task.ID.String(),
			Spec: task.Spec.Data(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRunnerFleetTaskComplete(w http.ResponseWriter, r *http.Request) {
	fleet, ok := s.authenticateRunnerFleet(w, r)
	if !ok {
		return
	}

	taskIDStr := strings.TrimSpace(mux.Vars(r)["taskId"])
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
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

	var req runners.FleetCompleteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	store := runners.NewPostgresStore()
	task, err := store.FindTask(taskID)
	if err != nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if task.FleetID != fleet.ID {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	wasTerminal := task.IsTerminal()
	task, err = store.CompleteJob(taskID, req)
	if err != nil {
		log.Errorf("runner fleet complete: %v", err)
		http.Error(w, "could not complete task", http.StatusInternalServerError)
		return
	}

	if wasTerminal {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := s.finishRunnerTask(task); err != nil {
		log.Errorf("runner fleet complete: finish execution: %v", err)
		http.Error(w, "could not finish execution", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) authenticateRunnerFleet(w http.ResponseWriter, r *http.Request) (*runners.RunnerFleet, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	store := runners.NewPostgresStore()
	fleet, err := store.FindFleetByAuthToken(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return fleet, true
}
