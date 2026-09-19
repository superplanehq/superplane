package public

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"gorm.io/gorm"
)

const (
	artifactRedirectTTL = 5 * time.Minute
	artifactCleanupTTL  = 10 * time.Second
)

type runnerArtifactResponse struct {
	ArtifactID  string `json:"artifact_id"`
	FileID      string `json:"file_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	PublicURL   string `json:"public_url"`
	Markdown    string `json:"markdown"`
}

type runnerArtifactContext struct {
	Factory       *models.Factory
	Order         *models.FactoryWorkOrder
	Canvas        *models.Canvas
	Node          *models.CanvasNode
	Line          *models.FactoryLine
	Execution     *models.FactoryWorkOrderExecution
	NodeExecution *models.CanvasNodeExecution
}

func (s *Server) handleRunnerArtifactUpload(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateArtifactRunner(w, r)
	if !ok {
		return
	}
	logger := log.WithFields(log.Fields{
		"artifact_type": "file",
		"work_order_id": scope.WorkOrderID,
		"run_id":        scope.CanvasRunID,
		"size_bytes":    r.ContentLength,
	})
	filename, contentType, reader, ok := validateArtifactUploadRequest(w, r)
	if !ok {
		logger.Warn("rejected runner artifact upload")
		return
	}

	db := database.DB(r.Context())
	artifactContext, err := loadRunnerArtifactContext(db, scope)
	if err != nil {
		logger.WithError(err).Warn("rejected runner artifact scope")
		http.Error(w, "Artifact upload scope is invalid", http.StatusForbidden)
		return
	}

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: scope.OrganizationID,
		FactoryID:      scope.FactoryID,
		WorkOrderID:    scope.WorkOrderID,
		Filename:       filename,
		ContentType:    contentType,
		Purpose:        models.FilePurposeArtifact,
	})
	if err != nil {
		logger.WithError(err).Warn("failed to create runner artifact file")
		writeArtifactUploadError(w, err)
		return
	}
	provider := blob.Current()
	if provider == nil {
		cleanupRunnerArtifact(nil, file, logger)
		logger.Error("failed to store runner artifact: blob storage is not configured")
		http.Error(w, "File storage is not configured", http.StatusInternalServerError)
		return
	}
	upload, err := storedfiles.StorePendingUpload(r.Context(), provider, file, reader)
	if err != nil {
		cleanupRunnerArtifact(provider, file, logger)
		logger.WithError(err).Warn("failed to store runner artifact")
		writeArtifactUploadError(w, err)
		return
	}
	if upload.SizeBytes != r.ContentLength {
		cleanupRunnerArtifact(provider, file, logger)
		logger.WithField("stored_size_bytes", upload.SizeBytes).Warn("rejected incomplete runner artifact upload")
		http.Error(w, "Artifact body does not match Content-Length", http.StatusBadRequest)
		return
	}

	publicURL, err := artifactPublicURL(file)
	if err != nil {
		cleanupRunnerArtifact(provider, file, logger)
		logger.WithError(err).Error("failed to create runner artifact URL")
		http.Error(w, "Failed to create artifact URL", http.StatusInternalServerError)
		return
	}
	title := decodeArtifactTitle(r.Header.Get("X-SuperPlane-Artifact-Title"))
	if title == "" {
		title = file.Filename
	}
	data := map[string]any{
		"fileId":      file.ID.String(),
		"filename":    file.Filename,
		"contentType": file.ContentType,
		"sizeBytes":   upload.SizeBytes,
		"title":       title,
		"url":         publicURL,
	}
	var artifact *models.FactoryWorkOrderArtifact
	err = db.Transaction(func(tx *gorm.DB) error {
		if readyErr := file.MarkReady(tx, upload.SizeBytes, upload.Checksum); readyErr != nil {
			return readyErr
		}
		artifact, err = artifactContext.Order.CreateArtifact(tx, models.FactoryWorkOrderArtifactParams{
			Type:       models.FactoryWorkOrderArtifactTypeFile,
			Data:       data,
			Automation: artifactAutomationRef(artifactContext),
			Run:        &factory.RunRef{ID: scope.CanvasRunID},
		})
		return err
	})
	if err != nil {
		cleanupRunnerArtifact(provider, file, logger)
		logger.WithError(err).Error("failed to attach runner artifact")
		http.Error(w, "Failed to attach artifact", http.StatusInternalServerError)
		return
	}

	if err := messages.PublishFactoryWorkOrderUpdated(scope.FactoryID.String(), scope.WorkOrderID.String(), factory.EventTypeOrderArtifactAdded); err != nil {
		log.WithError(err).Warn("failed to publish runner artifact update")
	}
	response := runnerArtifactResponse{
		ArtifactID:  artifact.ID.String(),
		FileID:      file.ID.String(),
		Filename:    file.Filename,
		ContentType: file.ContentType,
		SizeBytes:   file.SizeBytes,
		PublicURL:   publicURL,
		Markdown:    artifactMarkdown(title, file.ContentType, publicURL),
	}
	logger.WithFields(log.Fields{
		"content_type": contentType,
		"file_id":      file.ID,
		"artifact_id":  artifact.ID,
	}).Info("stored runner artifact")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Warn("failed to encode runner artifact response")
	}
}

func cleanupRunnerArtifact(provider blob.Provider, file *models.File, logger *log.Entry) {
	cleanupContext, cancel := context.WithTimeout(context.Background(), artifactCleanupTTL)
	defer cancel()
	cleanupDB := database.DB(cleanupContext)
	var err error
	if provider == nil {
		err = file.Delete(cleanupDB)
	} else {
		err = storedfiles.DeleteObjectAndRow(cleanupContext, cleanupDB, provider, file)
	}
	if err != nil {
		logger.WithError(err).Warn("failed to clean up rejected runner artifact")
	}
}

func decodeArtifactTitle(value string) string {
	trimmed := strings.TrimSpace(value)
	decoded, err := url.PathUnescape(trimmed)
	if err != nil {
		return trimmed
	}
	return strings.TrimSpace(decoded)
}

func (s *Server) authenticateArtifactRunner(w http.ResponseWriter, r *http.Request) (*runneraction.ArtifactUploadScope, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return nil, false
	}
	scope, err := runneraction.ParseArtifactUploadToken(s.jwt, strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
	if err != nil {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return nil, false
	}
	return scope, true
}

func validateArtifactUploadRequest(w http.ResponseWriter, r *http.Request) (string, string, io.Reader, bool) {
	if r.ContentLength <= 0 {
		http.Error(w, "Content-Length is required", http.StatusLengthRequired)
		return "", "", nil, false
	}
	if r.ContentLength > int64(models.MaxArtifactFileBytes) {
		http.Error(w, "Artifact exceeds the 100 MiB limit", http.StatusRequestEntityTooLarge)
		return "", "", nil, false
	}
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
	if err != nil || strings.TrimSpace(params["filename"]) == "" {
		http.Error(w, "Content-Disposition with a filename is required", http.StatusBadRequest)
		return "", "", nil, false
	}
	filename := filepath.Base(strings.TrimSpace(params["filename"]))
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !models.IsAllowedArtifactContentType(contentType) {
		http.Error(w, "Artifact content type is not supported", http.StatusUnsupportedMediaType)
		return "", "", nil, false
	}
	if !artifactFilenameMatchesContentType(filename, contentType) {
		http.Error(w, "Artifact filename does not match its content type", http.StatusUnsupportedMediaType)
		return "", "", nil, false
	}
	reader := bufio.NewReader(r.Body)
	prefix, _ := reader.Peek(16)
	if !artifactSignatureMatches(contentType, prefix) {
		http.Error(w, "Artifact content does not match its content type", http.StatusUnsupportedMediaType)
		return "", "", nil, false
	}
	return filename, contentType, reader, true
}

func artifactFilenameMatchesContentType(filename, contentType string) bool {
	extension := strings.ToLower(filepath.Ext(filename))
	switch contentType {
	case "image/png":
		return extension == ".png"
	case "image/jpeg":
		return extension == ".jpg" || extension == ".jpeg"
	case "image/webp":
		return extension == ".webp"
	case "video/webm":
		return extension == ".webm"
	case "video/mp4":
		return extension == ".mp4"
	default:
		return false
	}
}

func artifactSignatureMatches(contentType string, prefix []byte) bool {
	switch contentType {
	case "image/png":
		return len(prefix) >= 8 && string(prefix[:8]) == "\x89PNG\r\n\x1a\n"
	case "image/jpeg":
		return len(prefix) >= 3 && prefix[0] == 0xff && prefix[1] == 0xd8 && prefix[2] == 0xff
	case "image/webp":
		return len(prefix) >= 12 && string(prefix[:4]) == "RIFF" && string(prefix[8:12]) == "WEBP"
	case "video/webm":
		return len(prefix) >= 4 && prefix[0] == 0x1a && prefix[1] == 0x45 && prefix[2] == 0xdf && prefix[3] == 0xa3
	case "video/mp4":
		return len(prefix) >= 8 && string(prefix[4:8]) == "ftyp"
	default:
		return false
	}
}

func loadRunnerArtifactContext(db *gorm.DB, scope *runneraction.ArtifactUploadScope) (*runnerArtifactContext, error) {
	nodeExecution, err := models.FindNodeExecutionInTransaction(db, scope.CanvasID, scope.NodeExecutionID)
	if err != nil || nodeExecution.RunID != scope.CanvasRunID || nodeExecution.NodeID != scope.NodeID {
		return nil, fmt.Errorf("node execution does not match token")
	}
	runContext, err := runneraction.ResolveArtifactRunContext(db, scope.CanvasRunID)
	if err != nil || runContext.OrganizationID != scope.OrganizationID || runContext.FactoryID != scope.FactoryID ||
		runContext.WorkOrderID != scope.WorkOrderID || runContext.CanvasID != scope.CanvasID {
		return nil, fmt.Errorf("artifact run context does not match token")
	}
	factoryModel, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		return nil, err
	}
	order, err := factoryModel.FindWorkOrder(db, scope.WorkOrderID)
	if err != nil {
		return nil, err
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, scope.CanvasID)
	if err != nil || canvas.OrganizationID != scope.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != scope.FactoryID {
		return nil, fmt.Errorf("canvas does not match token")
	}
	node, err := models.FindCanvasNode(db, scope.CanvasID, scope.NodeID)
	if err != nil {
		return nil, err
	}
	var line *models.FactoryLine
	if runContext.LineExecution != nil {
		line, _ = factoryModel.FindLine(db, runContext.LineExecution.LineID)
	}
	return &runnerArtifactContext{
		Factory:       factoryModel,
		Order:         order,
		Canvas:        canvas,
		Node:          node,
		Line:          line,
		Execution:     runContext.LineExecution,
		NodeExecution: nodeExecution,
	}, nil
}

func artifactAutomationRef(context *runnerArtifactContext) *factory.AutomationRef {
	ref := &factory.AutomationRef{
		NodeID:   context.Node.NodeID,
		NodeName: context.Node.Name,
		AppID:    context.Canvas.ID,
		AppName:  context.Canvas.Name,
	}
	if context.Execution == nil {
		return ref
	}
	stepIndex := context.Execution.StepIndex
	ref.LineID = context.Execution.LineID
	ref.StepIndex = &stepIndex
	ref.StepName = context.Execution.StepName
	if context.Line != nil {
		ref.LineName = context.Line.Name
	}
	return ref
}

func artifactPublicURL(file *models.File) (string, error) {
	if file.PublicID == nil || *file.PublicID == uuid.Nil {
		return "", fmt.Errorf("artifact public id is missing")
	}
	return fmt.Sprintf("%s/api/v1/public/artifacts/%s/%s", blob.PublicBaseURL(), file.PublicID, url.PathEscape(file.Filename)), nil
}

func artifactMarkdown(title, contentType, publicURL string) string {
	label := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]").Replace(title)
	if strings.HasPrefix(contentType, "image/") {
		return fmt.Sprintf("![%s](%s)", label, publicURL)
	}
	return fmt.Sprintf("[Watch %s](%s)", label, publicURL)
}

func writeArtifactUploadError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, models.ErrFileQuotaExceeded) {
		status = http.StatusRequestEntityTooLarge
	} else if errors.Is(err, models.ErrFileContentType) {
		status = http.StatusUnsupportedMediaType
	}
	http.Error(w, err.Error(), status)
}

func (s *Server) handlePublicArtifactDownload(w http.ResponseWriter, r *http.Request) {
	publicID, err := uuid.Parse(strings.TrimSpace(mux.Vars(r)["public_id"]))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err := models.FindReadyArtifactFileByPublicID(database.DB(r.Context()), publicID)
	if err != nil {
		if !errors.Is(err, models.ErrFileNotFound) {
			log.WithError(err).Error("failed to load public artifact")
		}
		http.NotFound(w, r)
		return
	}
	if mux.Vars(r)["filename"] != file.Filename {
		http.NotFound(w, r)
		return
	}
	provider := blob.Current()
	if provider == nil {
		http.Error(w, "File storage is not configured", http.StatusInternalServerError)
		return
	}
	servePublicArtifact(w, r, provider, file)
}

func servePublicArtifact(w http.ResponseWriter, r *http.Request, provider blob.Provider, file *models.File) {
	setArtifactDownloadHeaders(w, file)
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
		w.WriteHeader(http.StatusOK)
		return
	}
	if provider.Name() == blob.ProviderGCS {
		signedURL, signErr := provider.SignedGetURL(r.Context(), file.StorageKey, artifactRedirectTTL)
		if signErr != nil {
			log.WithError(signErr).Error("failed to sign public artifact URL")
			http.Error(w, "Failed to download artifact", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, signedURL, http.StatusTemporaryRedirect)
		return
	}
	reader, err := provider.Get(r.Context(), file.StorageKey)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer reader.Close()
	if seeker, ok := reader.(io.ReadSeeker); ok {
		http.ServeContent(w, r, file.Filename, file.UpdatedAt, seeker)
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	if _, err := io.Copy(w, reader); err != nil {
		log.WithError(err).Warn("failed to stream public artifact")
	}
}

func setArtifactDownloadHeaders(w http.ResponseWriter, file *models.File) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": file.Filename}))
}
