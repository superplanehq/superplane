package public

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

func (s *Server) handleFileContentUpload(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}

	fileID, err := parseFileIDParam(r)
	if err != nil {
		http.Error(w, "Invalid file id", http.StatusBadRequest)
		return
	}

	db := database.DB(r.Context())
	file, err := models.FindFile(db, fileID)
	if err != nil {
		if errors.Is(err, models.ErrFileNotFound) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		log.Errorf("Failed to load file %s: %v", fileID, err)
		http.Error(w, "Failed to load file", http.StatusInternalServerError)
		return
	}

	if file.CreatedByID == nil || *file.CreatedByID != user.ID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	if file.OrganizationID == nil || *file.OrganizationID != user.OrganizationID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	resource, action := fileUploadPermission(file.Scope)
	allowed, err := s.authService.CheckOrganizationPermission(
		r.Context(),
		user.ID.String(),
		user.OrganizationID.String(),
		resource,
		action,
	)
	if err != nil {
		log.Errorf("Failed to check file upload permission: %v", err)
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	if !allowed {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := storedfiles.CompleteUpload(r.Context(), db, blob.Current(), file, r.Body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrFileQuotaExceeded) {
			status = http.StatusRequestEntityTooLarge
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublicFileDownload(w http.ResponseWriter, r *http.Request) {
	fileID, err := parseFileIDParam(r)
	if err != nil {
		http.Error(w, "Invalid file id", http.StatusBadRequest)
		return
	}

	expires, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("expires")), 10, 64)
	if err != nil {
		http.Error(w, "Invalid file URL", http.StatusBadRequest)
		return
	}
	sig := strings.TrimSpace(r.URL.Query().Get("sig"))
	key, err := blob.SigningKey()
	if err != nil {
		log.Errorf("Failed to load file signing key: %v", err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
	if err := blob.VerifyFileAccess(fileID, expires, sig, key); err != nil {
		http.Error(w, "Invalid file URL", http.StatusForbidden)
		return
	}

	db := database.DB(r.Context())
	file, err := models.FindFile(db, fileID)
	if err != nil {
		if errors.Is(err, models.ErrFileNotFound) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		log.Errorf("Failed to load file %s: %v", fileID, err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
	if file.State != models.FileStateReady {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	provider := blob.Current()
	if provider == nil {
		http.Error(w, "File storage is not configured", http.StatusInternalServerError)
		return
	}
	reader, err := provider.Get(r.Context(), file.StorageKey)
	if err != nil {
		if errors.Is(err, blob.ErrNotFound) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		log.Errorf("Failed to read file %s: %v", fileID, err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	contentType := file.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": file.Filename,
	}))
	if file.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	}

	if _, err := io.Copy(w, reader); err != nil {
		log.Errorf("Failed to copy file %s: %v", fileID, err)
	}
}

func parseFileIDParam(r *http.Request) (uuid.UUID, error) {
	raw := mux.Vars(r)["file_id"]
	if raw == "" {
		return uuid.Nil, fmt.Errorf("file id is required")
	}
	return uuid.Parse(raw)
}

func fileUploadPermission(scope string) (resource, action string) {
	if scope == blob.ScopeTask {
		return "work_orders", "update"
	}
	return "factories", "update"
}
