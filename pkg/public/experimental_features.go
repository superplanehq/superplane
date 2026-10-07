package public

import (
	"encoding/json"
	"net/http"

	"github.com/superplanehq/superplane/pkg/features"
)

type experimentalFeatureItem struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Released    bool   `json:"released"`
}

func experimentalFeatureItems() []experimentalFeatureItem {
	registry := features.All()
	items := make([]experimentalFeatureItem, 0, len(registry))
	for _, f := range registry {
		items = append(items, experimentalFeatureItem{
			ID:          f.ID,
			Label:       f.Label,
			Description: f.Description,
			Released:    f.Released != nil && *f.Released,
		})
	}
	return items
}

// listExperimentalFeatures returns the static registry of experimental
// features available in this installation. It is account-authenticated and
// does not require installation admin: any signed-in user can fetch it so
// the UI can decide whether to render gated experiences. Per-organization
// enablement is exposed separately on the organization resource for members,
// and on the admin experimental-features route for installation admins.
func (s *Server) listExperimentalFeatures(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"features": experimentalFeatureItems(),
	})
}
