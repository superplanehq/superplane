package api

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

type orgFleetOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrgListFleets lists runner fleets for canvas configuration (machine type dropdown).
// Requires organization auth; returns an empty list when the runner feature is disabled for the org.
func (h *Handler) OrgListFleets(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	enabled, err := models.HasExperimentalFeature(user.OrganizationID, features.FeatureRunner)
	if err != nil {
		log.Errorf("runner fleets: check experimental feature: %v", err)
		http.Error(w, "Failed to list machine types", http.StatusInternalServerError)
		return
	}

	items := []orgFleetOption{}
	if !enabled {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
		return
	}

	fleets, err := h.store().ListFleets()
	if err != nil {
		log.Errorf("runner fleets: list: %v", err)
		http.Error(w, "Failed to list machine types", http.StatusInternalServerError)
		return
	}

	items = make([]orgFleetOption, 0, len(fleets))
	for _, f := range fleets {
		items = append(items, orgFleetOption{
			ID:   f.ID.String(),
			Name: f.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
