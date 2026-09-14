package public

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type organizationFactorySettingsResponse struct {
	MaxParallelFactoryTasks *int `json:"max_parallel_factory_tasks"`
	InstallationDefault     int  `json:"installation_default"`
	Effective               int  `json:"effective"`
}

type organizationFactorySettingsRequest struct {
	MaxParallelFactoryTasks *int `json:"max_parallel_factory_tasks"`
}

func (s *Server) adminGetOrganizationFactorySettings(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	response, err := describeOrganizationFactorySettings(orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization factory settings: %v", err)
		http.Error(w, "Failed to load factory settings", http.StatusInternalServerError)
		return
	}

	respondJSON(w, response)
}

func (s *Server) adminUpdateOrganizationFactorySettings(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	var req organizationFactorySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.SetOrganizationMaxParallelFactoryTasks(database.Conn(), orgID, req.MaxParallelFactoryTasks); err != nil {
		if errors.Is(err, models.ErrMaxParallelFactoryTasksInvalid) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Errorf("admin: failed to update organization factory settings: %v", err)
		http.Error(w, "Failed to update factory settings", http.StatusInternalServerError)
		return
	}

	response, err := describeOrganizationFactorySettings(orgID)
	if err != nil {
		log.Errorf("admin: failed to load organization factory settings: %v", err)
		http.Error(w, "Failed to load factory settings", http.StatusInternalServerError)
		return
	}

	respondJSON(w, response)
}

func describeOrganizationFactorySettings(orgID uuid.UUID) (organizationFactorySettingsResponse, error) {
	org, err := models.FindOrganizationByID(orgID.String())
	if err != nil {
		return organizationFactorySettingsResponse{}, err
	}

	installation, err := models.GetInstallationMetadata(database.Conn())
	if err != nil {
		return organizationFactorySettingsResponse{}, err
	}

	effective, err := models.ResolveOrganizationFactoryMaxParallelTasks(database.Conn(), orgID)
	if err != nil {
		return organizationFactorySettingsResponse{}, err
	}

	return organizationFactorySettingsResponse{
		MaxParallelFactoryTasks: org.MaxParallelFactoryTasks,
		InstallationDefault:     installation.MaxParallelFactoryTasks,
		Effective:               effective,
	}, nil
}
