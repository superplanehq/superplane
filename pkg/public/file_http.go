package public

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	if r.ContentLength > models.MaxFileBytes {
		http.Error(w, fmt.Sprintf("file exceeds %d bytes", models.MaxFileBytes), http.StatusRequestEntityTooLarge)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)
	r.Body = http.MaxBytesReader(w, r.Body, models.MaxFileBytes)

	if err := storedfiles.CompleteUpload(r.Context(), db, blob.Current(), file, r.Body); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, models.ErrFileQuotaExceeded) || isMaxBytesError(err) {
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
	w.Header().Set("Accept-Ranges", "bytes")

	if seeker, ok := reader.(io.ReadSeeker); ok {
		http.ServeContent(w, r, file.Filename, file.UpdatedAt, seeker)
		return
	}
	if file.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	}
	if strings.TrimSpace(r.Header.Get("Range")) == "" {
		if _, err := io.Copy(w, reader); err != nil {
			log.Errorf("Failed to copy file %s: %v", fileID, err)
		}
		return
	}
	_ = reader.Close()
	start, length, ok := parseBytesRange(r.Header.Get("Range"), file.SizeBytes)
	if !ok {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", file.SizeBytes))
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}
	rangeReader, err := provider.GetRange(r.Context(), file.StorageKey, start, length)
	if err != nil {
		log.Errorf("Failed to read file range %s: %v", fileID, err)
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
	defer rangeReader.Close()
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+length-1, file.SizeBytes))
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(http.StatusPartialContent)
	if _, err := io.Copy(w, rangeReader); err != nil {
		log.Errorf("Failed to copy file range %s: %v", fileID, err)
	}
}

func parseBytesRange(header string, size int64) (start, length int64, ok bool) {
	header = strings.TrimSpace(header)
	if size <= 0 || !strings.HasPrefix(header, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimPrefix(header, "bytes=")
	if strings.Contains(spec, ",") {
		return 0, 0, false
	}
	parts := strings.Split(spec, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	if parts[0] == "" {
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false
		}
		if suffix > size {
			suffix = size
		}
		return size - suffix, suffix, true
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, false
	}
	end := size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, false
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end - start + 1, true
}

func isMaxBytesError(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
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
