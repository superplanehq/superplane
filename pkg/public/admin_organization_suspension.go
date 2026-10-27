package public

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// These handlers are registered only on the installation-admin router.
func (s *Server) adminSuspendOrganization(w http.ResponseWriter, r *http.Request) {
	admin, id, ok := adminOrganizationPinRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&body); err != nil {
		http.Error(w, "Enter a suspension reason.", http.StatusBadRequest)
		return
	}
	organization := models.Organization{ID: id}
	err := organization.Suspend(database.DB(r.Context()), admin.ID, body.Reason)
	if organizationSuspensionError(w, err) {
		return
	}
	respondJSON(w, map[string]string{"status": "suspended"})
}

func (s *Server) adminReactivateOrganization(w http.ResponseWriter, r *http.Request) {
	admin, id, ok := adminOrganizationPinRequest(w, r)
	if !ok {
		return
	}
	organization := models.Organization{ID: id}
	if organizationSuspensionError(w, organization.Reactivate(database.DB(r.Context()), admin.ID)) {
		return
	}
	respondJSON(w, map[string]string{"status": "active"})
}

func (s *Server) adminOrganizationSuspensionHistory(w http.ResponseWriter, r *http.Request) {
	_, id, ok := adminOrganizationPinRequest(w, r)
	if !ok {
		return
	}
	events, err := models.ListOrganizationSuspensionEvents(database.DB(r.Context()), id)
	if organizationSuspensionError(w, err) {
		return
	}
	respondJSON(w, events)
}

func organizationSuspensionError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, models.ErrSuspensionReasonRequired):
		http.Error(w, models.SuspensionReasonRequiredMessage, http.StatusBadRequest)
	case errors.Is(err, models.ErrSuspensionCancellationPending):
		http.Error(w, models.SuspensionCancellationPendingMessage, http.StatusConflict)
	case errors.Is(err, gorm.ErrRecordNotFound):
		http.Error(w, "Organization not found.", http.StatusNotFound)
	default:
		http.Error(w, "Could not change organization suspension.", http.StatusInternalServerError)
	}
	return true
}
