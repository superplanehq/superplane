package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

type adminIntakeCatalogEntry struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	Status        string `json:"status"`
	StatusNote    string `json:"status_note"`
	Implemented   bool   `json:"implemented"`
	Deletable     bool   `json:"deletable"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	UpdatedByName string `json:"updated_by_name"`
}

type adminCreateIntakeCatalogEntryRequest struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	StatusNote string `json:"status_note"`
}

type adminUpdateIntakeCatalogEntryRequest struct {
	Name       *string `json:"name"`
	Category   *string `json:"category"`
	Status     *string `json:"status"`
	StatusNote *string `json:"status_note"`
}

func (s *Server) adminGetIntakeCatalogEntry(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	s.respondIntakeCatalogEntry(w, db, mux.Vars(r)["key"])
}

func (s *Server) adminListIntakeCatalog(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	entries, err := models.ListIntakeCatalogEntries(db)
	if err != nil {
		log.Errorf("admin: failed to list intake catalog: %v", err)
		http.Error(w, "Failed to list intakes", http.StatusInternalServerError)
		return
	}

	items := make([]adminIntakeCatalogEntry, 0, len(entries))
	for i := range entries {
		items = append(items, serializeAdminIntakeCatalogEntry(&entries[i]))
	}
	respondJSON(w, map[string]any{"entries": items})
}

func (s *Server) adminCreateIntakeCatalogEntry(w http.ResponseWriter, r *http.Request) {
	var req adminCreateIntakeCatalogEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entry, err := models.NewIntakeCatalogEntry(req.Key, req.Name, req.Category, req.StatusNote)
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	db := database.DB(r.Context())
	if err := models.CreateIntakeCatalogEntry(db, entry, adminAccountID(r)); err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	s.respondIntakeCatalogEntry(w, db, entry.Key)
}

func (s *Server) adminUpdateIntakeCatalogEntry(w http.ResponseWriter, r *http.Request) {
	var req adminUpdateIntakeCatalogEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	entry, err := models.FindIntakeCatalogEntry(db, mux.Vars(r)["key"])
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	patch := models.IntakeCatalogPatch{
		Name:       req.Name,
		Category:   req.Category,
		Status:     req.Status,
		StatusNote: req.StatusNote,
	}
	if err := entry.Update(db, patch, adminAccountID(r)); err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	s.respondIntakeCatalogEntry(w, db, entry.Key)
}

func (s *Server) adminDeleteIntakeCatalogEntry(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	entry, err := models.FindIntakeCatalogEntry(db, mux.Vars(r)["key"])
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	if err := entry.Delete(db); err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) respondIntakeCatalogEntry(w http.ResponseWriter, db *gorm.DB, key string) {
	entry, err := models.FindIntakeCatalogEntry(db, key)
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}
	respondJSON(w, serializeAdminIntakeCatalogEntry(entry))
}

func serializeAdminIntakeCatalogEntry(entry *models.IntakeCatalogEntry) adminIntakeCatalogEntry {
	return adminIntakeCatalogEntry{
		Key:           entry.Key,
		Name:          entry.Name,
		Category:      entry.Category,
		Status:        entry.Status,
		StatusNote:    entry.StatusNote,
		Implemented:   entry.Implemented(),
		Deletable:     entry.Deletable(),
		CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     entry.UpdatedAt.UTC().Format(time.RFC3339),
		UpdatedByName: entry.UpdatedByName,
	}
}

func adminAccountID(r *http.Request) *uuid.UUID {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		return nil
	}
	return &account.ID
}

func respondIntakeCatalogError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrIntakeCatalogEntryNotFound):
		http.Error(w, "Intake not found", http.StatusNotFound)
	case errors.Is(err, models.ErrIntakeCatalogEntryExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, models.ErrIntakeCatalogKeyInvalid),
		errors.Is(err, models.ErrIntakeCatalogNameRequired),
		errors.Is(err, models.ErrIntakeCatalogCategoryInvalid),
		errors.Is(err, models.ErrIntakeCatalogStatusInvalid),
		errors.Is(err, models.ErrIntakeCatalogNotImplemented),
		errors.Is(err, models.ErrIntakeCatalogDeleteImplemented):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.Errorf("admin: intake catalog request failed: %v", err)
		http.Error(w, "Intake catalog request failed", http.StatusInternalServerError)
	}
}
