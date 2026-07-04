package public

import (
	"encoding/json"
	"net/http"

	"github.com/superplanehq/superplane/pkg/features"
)

// listExperimentalFeatures returns the static registry of experimental
// features available in this installation. It is account-authenticated and
// does not require installation admin: any signed-in user can fetch it so
// the UI can decide whether to render gated experiences. Per-organization
// enablement is exposed separately on the organization resource.
func (s *Server) listExperimentalFeatures(w http.ResponseWriter, r *http.Request) {
	type featureItem struct {
		ID          string `json:"id"`
		Label       string `json:"label"`
		Description string `json:"description"`
		Released    bool   `json:"released"`
	}

	registry := features.All()
	items := make([]featureItem, 0, len(registry))
	for _, f := range registry {
		items = append(items, featureItem{
			ID:          f.ID,
			Label:       f.Label,
			Description: f.Description,
			Released:    f.Released != nil && *f.Released,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"features": items,
	})
}
