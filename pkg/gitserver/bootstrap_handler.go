package gitserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// RegisterBootstrapRoute adds POST /git/{slug}/bootstrap to init a repo from canvas state.
func (s *Server) RegisterBootstrapRoute(router *mux.Router, baseURL string, registry *Registry) {
	router.HandleFunc("/git/{slug}/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticate(w, r) {
			return
		}

		slug := mux.Vars(r)["slug"]

		var body struct {
			CanvasID string `json:"canvasId"`
			OrgID    string `json:"orgId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		if body.CanvasID == "" || body.OrgID == "" {
			http.Error(w, "canvasId and orgId are required", http.StatusBadRequest)
			return
		}

		// Get the token from Basic auth
		_, token, _ := r.BasicAuth()

		log.Infof("gitserver: bootstrapping repo %s for canvas %s", slug, body.CanvasID)

		err := s.BootstrapFromAPI(slug, body.CanvasID, body.OrgID, baseURL, token)
		if err != nil {
			log.Errorf("gitserver: bootstrap failed for %s: %v", slug, err)
			http.Error(w, fmt.Sprintf("Bootstrap failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Register the slug in the in-memory registry
		registry.Register(slug, &SlugToCanvasMapping{
			CanvasID: body.CanvasID,
			OrgID:    body.OrgID,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"slug":    slug,
			"repoURL": fmt.Sprintf("%s/git/%s", baseURL, slug),
		})
	}).Methods("POST")
}
