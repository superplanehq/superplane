package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// runnerArtifactView describes one artifact for list_artifacts and
// get_artifact. Content fields are set only by get_artifact.
type runnerArtifactView struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	CreatedAt   string `json:"created_at"`
	Body        string `json:"body,omitempty"`
	PublicURL   string `json:"public_url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
}

var runnerArtifactListTypes = []string{
	models.FactoryWorkOrderArtifactTypeMarkdown,
	models.FactoryWorkOrderArtifactTypeLink,
	models.FactoryWorkOrderArtifactTypeFile,
}

func (s *Server) handleRunnerListArtifacts(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeArtifactsRead)(w, r)
	if !ok {
		return
	}
	artifactType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if artifactType != "" && !slices.Contains(runnerArtifactListTypes, artifactType) {
		http.Error(w, "type must be markdown, link, or file", http.StatusBadRequest)
		return
	}

	artifacts, err := target.order.ListArtifacts(database.DB(r.Context()))
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}
	views := make([]runnerArtifactView, 0, len(artifacts))
	for i := range artifacts {
		if artifactType != "" && artifacts[i].Type != artifactType {
			continue
		}
		views = append(views, runnerArtifactViewFrom(target.order.ID, &artifacts[i], false))
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"artifacts": views})
}

func (s *Server) handleRunnerGetArtifact(w http.ResponseWriter, r *http.Request) {
	target, ok := s.runnerOwnerTarget(runneraction.ScopeArtifactsRead)(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	rawID := strings.TrimSpace(query.Get("id"))
	key := strings.TrimSpace(query.Get("key"))
	if (rawID == "") == (key == "") {
		http.Error(w, "give one of id or key", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	var artifact *models.FactoryWorkOrderArtifact
	var err error
	if rawID != "" {
		id, parseErr := uuid.Parse(rawID)
		if parseErr != nil {
			http.Error(w, "Artifact not found", http.StatusNotFound)
			return
		}
		artifact, err = target.order.FindArtifactByID(db, id)
	} else {
		artifact, err = findRunnerArtifactByKey(db, target.order, key)
	}
	if err != nil {
		writeRunnerTaskError(w, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, runnerArtifactViewFrom(target.order.ID, artifact, true))
}

// findRunnerArtifactByKey finds an artifact by the key that a run gave it,
// then by its stored key. Runs store "<task id>:<key>".
func findRunnerArtifactByKey(db *gorm.DB, order *models.FactoryWorkOrder, key string) (*models.FactoryWorkOrderArtifact, error) {
	scoped := scopedArtifactLinkKey(order.ID, key)
	if len(scoped) <= models.MaxFactoryWorkOrderArtifactKeyBytes {
		artifact, err := order.FindArtifactByKey(db, scoped)
		if !errors.Is(err, models.ErrFactoryWorkOrderArtifactNotFound) {
			return artifact, err
		}
	}
	return order.FindArtifactByKey(db, key)
}

// runnerArtifactKey returns the key as the run gave it.
func runnerArtifactKey(orderID uuid.UUID, artifact *models.FactoryWorkOrderArtifact) string {
	if artifact.Key == nil {
		return ""
	}
	return strings.TrimPrefix(*artifact.Key, orderID.String()+":")
}

func runnerArtifactViewFrom(orderID uuid.UUID, artifact *models.FactoryWorkOrderArtifact, withContent bool) runnerArtifactView {
	data := map[string]any{}
	_ = json.Unmarshal(artifact.Data, &data)
	view := runnerArtifactView{
		ID:        artifact.ID.String(),
		Key:       runnerArtifactKey(orderID, artifact),
		Type:      artifact.Type,
		Title:     artifactTitle(artifact.Data),
		URL:       runnerArtifactField(data, "url"),
		CreatedAt: artifact.CreatedAt.UTC().Format(time.RFC3339),
	}
	if !withContent {
		return view
	}
	switch artifact.Type {
	case models.FactoryWorkOrderArtifactTypeMarkdown:
		view.Body = runnerArtifactField(data, "body")
	case models.FactoryWorkOrderArtifactTypeFile:
		view.PublicURL = view.URL
		view.ContentType = runnerArtifactField(data, "contentType")
		if size, ok := data["sizeBytes"].(float64); ok {
			view.SizeBytes = int64(size)
		}
	}
	return view
}

func runnerArtifactField(data map[string]any, name string) string {
	value, _ := data[name].(string)
	return strings.TrimSpace(value)
}
