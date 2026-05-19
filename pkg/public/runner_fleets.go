package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/runners"
	"gorm.io/gorm"
)

type registerFleetRequest struct {
	Name      string   `json:"name"`
	FleetURL  string   `json:"fleet_url"`
	AuthToken string   `json:"auth_token"`
	Labels    []string `json:"labels"`
}

type fleetResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	FleetURL  string   `json:"fleet_url"`
	Labels    []string `json:"labels"`
	CreatedAt string   `json:"created_at,omitempty"`
}

func fleetToResponse(f runners.RunnerFleet) fleetResponse {
	labels := f.Labels.Data()
	if labels == nil {
		labels = []string{}
	}
	r := fleetResponse{
		ID:       f.ID.String(),
		Name:     f.Name,
		FleetURL: f.FleetURL,
		Labels:   labels,
	}
	if f.CreatedAt != nil {
		r.CreatedAt = f.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return r
}

// adminRegisterFleet registers a new runner fleet.
func (s *Server) adminRegisterFleet(w http.ResponseWriter, r *http.Request) {
	var req registerFleetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.FleetURL) == "" {
		http.Error(w, "fleet_url is required", http.StatusBadRequest)
		return
	}

	store := runners.NewPostgresStore()
	fleet, err := store.CreateFleet(req.Name, req.FleetURL, req.AuthToken, req.Labels)
	if err != nil {
		log.Errorf("admin: failed to register runner fleet: %v", err)
		http.Error(w, "Failed to register fleet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fleetToResponse(*fleet))
}

// adminListFleets lists all registered runner fleets.
func (s *Server) adminListFleets(w http.ResponseWriter, r *http.Request) {
	store := runners.NewPostgresStore()
	fleets, err := store.ListFleets()
	if err != nil {
		log.Errorf("admin: failed to list runner fleets: %v", err)
		http.Error(w, "Failed to list fleets", http.StatusInternalServerError)
		return
	}

	items := make([]fleetResponse, 0, len(fleets))
	for _, f := range fleets {
		items = append(items, fleetToResponse(f))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// adminDeleteFleet deletes a registered runner fleet by ID.
func (s *Server) adminDeleteFleet(w http.ResponseWriter, r *http.Request) {
	fleetIDStr := mux.Vars(r)["fleetId"]
	fleetID, err := uuid.Parse(fleetIDStr)
	if err != nil {
		http.Error(w, "Fleet not found", http.StatusNotFound)
		return
	}

	store := runners.NewPostgresStore()
	if err := store.DeleteFleet(fleetID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Fleet not found", http.StatusNotFound)
			return
		}
		log.Errorf("admin: failed to delete runner fleet %s: %v", fleetID, err)
		http.Error(w, "Failed to delete fleet", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
