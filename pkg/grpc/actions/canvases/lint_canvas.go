package canvases

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/linter"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

// jsonError writes a JSON error response with the given status code.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// LintCanvasHandler returns an http.HandlerFunc that lints a canvas by ID.
// It reads the canvas spec from the live version and runs the linter.
//
// Route: POST /api/v1/canvases/{canvasId}/lint
func LintCanvasHandler(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract organization ID from header (set by auth middleware).
		orgID := r.Header.Get("X-Organization-Id")
		if orgID == "" {
			jsonError(w, "missing organization id", http.StatusUnauthorized)
			return
		}

		// Extract canvas ID from gorilla/mux route variables.
		canvasID := mux.Vars(r)["canvasId"]
		if canvasID == "" {
			jsonError(w, "missing canvas id", http.StatusBadRequest)
			return
		}

		orgUUID, err := uuid.Parse(orgID)
		if err != nil {
			jsonError(w, "invalid organization id", http.StatusBadRequest)
			return
		}

		canvasUUID, err := uuid.Parse(canvasID)
		if err != nil {
			jsonError(w, "invalid canvas id", http.StatusBadRequest)
			return
		}

		// Load the canvas.
		canvas, err := models.FindCanvas(orgUUID, canvasUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				jsonError(w, "canvas not found", http.StatusNotFound)
				return
			}
			log.WithError(err).Error("failed to find canvas")
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Load the live version.
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), canvas)
		if err != nil {
			log.WithError(err).Error("failed to find live canvas version")
			jsonError(w, "failed to load canvas version", http.StatusInternalServerError)
			return
		}

		// Run the linter.
		nodes := []models.Node(version.Nodes)
		edges := []models.Edge(version.Edges)
		result := linter.LintCanvas(nodes, edges, reg)

		// Return JSON response.
		w.Header().Set("Content-Type", "application/json")
		if result.Status == "fail" {
			w.WriteHeader(http.StatusUnprocessableEntity)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.WithError(err).Error("failed to encode lint result")
		}
	}
}
