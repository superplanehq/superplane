package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

type adminIntakeCatalogOrganization struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	AddedAt string `json:"added_at"`
}

type adminIntakeCatalogEntry struct {
	Key           string                           `json:"key"`
	Name          string                           `json:"name"`
	Category      string                           `json:"category"`
	Status        string                           `json:"status"`
	StatusNote    string                           `json:"status_note"`
	EnabledForAll bool                             `json:"enabled_for_all"`
	Implemented   bool                             `json:"implemented"`
	Deletable     bool                             `json:"deletable"`
	CreatedAt     string                           `json:"created_at"`
	UpdatedAt     string                           `json:"updated_at"`
	UpdatedByName string                           `json:"updated_by_name"`
	Organizations []adminIntakeCatalogOrganization `json:"organizations"`
}

type adminCreateIntakeCatalogEntryRequest struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	StatusNote string `json:"status_note"`
}

type adminUpdateIntakeCatalogEntryRequest struct {
	Name          *string `json:"name"`
	Category      *string `json:"category"`
	Status        *string `json:"status"`
	StatusNote    *string `json:"status_note"`
	EnabledForAll *bool   `json:"enabled_for_all"`
}

type adminIntakeCatalogPreview struct {
	Key               string `json:"key"`
	Status            string `json:"status"`
	OrganizationAdded bool   `json:"organization_added"`
	Available         bool   `json:"available"`
}

func (s *Server) adminListIntakeCatalog(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	entries, err := models.ListIntakeCatalogEntries(db)
	if err != nil {
		log.Errorf("admin: failed to list intake catalog: %v", err)
		http.Error(w, "Failed to list intakes", http.StatusInternalServerError)
		return
	}

	access, err := models.ListIntakeCatalogOrganizations(db)
	if err != nil {
		log.Errorf("admin: failed to list intake catalog organizations: %v", err)
		http.Error(w, "Failed to list intakes", http.StatusInternalServerError)
		return
	}

	items := make([]adminIntakeCatalogEntry, 0, len(entries))
	for i := range entries {
		items = append(items, serializeAdminIntakeCatalogEntry(&entries[i], access))
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
		Name:          req.Name,
		Category:      req.Category,
		Status:        req.Status,
		StatusNote:    req.StatusNote,
		EnabledForAll: req.EnabledForAll,
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

func (s *Server) adminAddIntakeCatalogOrganization(w http.ResponseWriter, r *http.Request) {
	s.changeIntakeCatalogOrganization(w, r, func(tx *gorm.DB, entry *models.IntakeCatalogEntry, orgID uuid.UUID) error {
		return entry.AddOrganization(tx, orgID)
	})
}

func (s *Server) adminRemoveIntakeCatalogOrganization(w http.ResponseWriter, r *http.Request) {
	s.changeIntakeCatalogOrganization(w, r, func(tx *gorm.DB, entry *models.IntakeCatalogEntry, orgID uuid.UUID) error {
		return entry.RemoveOrganization(tx, orgID)
	})
}

// adminPreviewIntakeCatalogEntry tells what one organization sees for an
// entry. Without organization_id it answers for a company without access.
func (s *Server) adminPreviewIntakeCatalogEntry(w http.ResponseWriter, r *http.Request) {
	db := database.DB(r.Context())
	entry, err := models.FindIntakeCatalogEntry(db, mux.Vars(r)["key"])
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	added := false
	if raw := strings.TrimSpace(r.URL.Query().Get("organization_id")); raw != "" {
		orgID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		added, err = entry.HasOrganization(db, orgID)
		if err != nil {
			log.Errorf("admin: failed to preview intake %s: %v", entry.Key, err)
			http.Error(w, "Failed to load preview", http.StatusInternalServerError)
			return
		}
	}

	respondJSON(w, adminIntakeCatalogPreview{
		Key:               entry.Key,
		Status:            entry.Status,
		OrganizationAdded: added,
		Available:         entry.AvailableTo(added),
	})
}

func (s *Server) changeIntakeCatalogOrganization(
	w http.ResponseWriter,
	r *http.Request,
	change func(tx *gorm.DB, entry *models.IntakeCatalogEntry, orgID uuid.UUID) error,
) {
	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}
	if _, err := models.FindOrganizationByID(orgID.String()); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	db := database.DB(r.Context())
	entry, err := models.FindIntakeCatalogEntry(db, vars["key"])
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	if err := change(db, entry, orgID); err != nil {
		respondIntakeCatalogError(w, err)
		return
	}

	s.respondIntakeCatalogEntry(w, db, entry.Key)
}

func (s *Server) respondIntakeCatalogEntry(w http.ResponseWriter, db *gorm.DB, key string) {
	entry, err := models.FindIntakeCatalogEntry(db, key)
	if err != nil {
		respondIntakeCatalogError(w, err)
		return
	}
	access, err := models.ListIntakeCatalogOrganizations(db)
	if err != nil {
		log.Errorf("admin: failed to list intake catalog organizations: %v", err)
		http.Error(w, "Failed to load intake", http.StatusInternalServerError)
		return
	}
	respondJSON(w, serializeAdminIntakeCatalogEntry(entry, access))
}

func serializeAdminIntakeCatalogEntry(
	entry *models.IntakeCatalogEntry,
	access []models.IntakeCatalogOrganizationAccess,
) adminIntakeCatalogEntry {
	organizations := []adminIntakeCatalogOrganization{}
	for _, row := range access {
		if row.EntryKey != entry.Key {
			continue
		}
		organizations = append(organizations, adminIntakeCatalogOrganization{
			ID:      row.OrganizationID.String(),
			Name:    row.OrganizationName,
			AddedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return adminIntakeCatalogEntry{
		Key:           entry.Key,
		Name:          entry.Name,
		Category:      entry.Category,
		Status:        entry.Status,
		StatusNote:    entry.StatusNote,
		EnabledForAll: entry.EnabledForAll,
		Implemented:   entry.Implemented(),
		Deletable:     entry.Deletable(),
		CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     entry.UpdatedAt.UTC().Format(time.RFC3339),
		UpdatedByName: entry.UpdatedByName,
		Organizations: organizations,
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
		errors.Is(err, models.ErrIntakeCatalogInternalForAll),
		errors.Is(err, models.ErrIntakeCatalogDeleteImplemented),
		errors.Is(err, models.ErrIntakeCatalogAccessNotSupported):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.Errorf("admin: intake catalog request failed: %v", err)
		http.Error(w, "Intake catalog request failed", http.StatusInternalServerError)
	}
}
