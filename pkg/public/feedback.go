package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/services"
)

var errFeedbackOrganization = errors.New("feedback organization is not available")

var publishSupportFeedback = func(message messages.SupportFeedbackRequestedMessage) error {
	return message.Publish()
}

var resolveFeedbackOrganization = func(r *http.Request, user *models.User) (string, string, error) {
	ref := strings.TrimSpace(r.Header.Get("x-organization-id"))
	if ref == "" {
		ref = strings.TrimSpace(r.URL.Query().Get("organization_id"))
	}
	if ref == "" || strings.TrimSpace(user.GetEmail()) == "" {
		return "", "", errFeedbackOrganization
	}

	db := database.DB(r.Context())
	organization, err := models.FindOrganizationByIDOrSlug(db, ref)
	if err != nil || organization == nil {
		return "", "", errFeedbackOrganization
	}

	member, err := models.FindActiveUserByEmailInTransaction(db, organization.ID.String(), user.GetEmail())
	if err != nil || member == nil || !member.IsHuman() {
		return "", "", errFeedbackOrganization
	}

	return organization.ID.String(), organization.Name, nil
}

func (s *Server) handleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthenticated", http.StatusUnauthorized)
		return
	}
	if !user.IsHuman() {
		writeFeedbackError(w, http.StatusForbidden, "Feedback is available only for user accounts.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(services.MaxSupportFeedbackRequestBytes))
	if err := parseFeedbackForm(r); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeFeedbackError(w, http.StatusRequestEntityTooLarge, "The feedback form is too large.")
			return
		}
		writeFeedbackError(w, http.StatusBadRequest, "SuperPlane could not read the feedback form.")
		return
	}

	category, err := services.NormalizeFeedbackCategory(r.FormValue("category"))
	if err != nil {
		writeFeedbackError(w, http.StatusBadRequest, "Select a feedback category.")
		return
	}
	details, err := services.NormalizeFeedbackDetails(r.FormValue("details"))
	if err != nil {
		writeFeedbackError(w, http.StatusBadRequest, feedbackDetailsErrorMessage(err))
		return
	}

	attachment, err := readFeedbackAttachment(r)
	if err != nil {
		writeFeedbackError(w, http.StatusBadRequest, feedbackAttachmentErrorMessage(err))
		return
	}

	organizationID, organizationName, err := resolveFeedbackOrganization(r, user)
	if err != nil {
		writeFeedbackError(w, http.StatusForbidden, "Feedback is available only for an organization you belong to.")
		return
	}

	message := messages.SupportFeedbackRequestedMessage{
		Category:         category,
		Details:          details,
		UserName:         user.Name,
		UserEmail:        user.GetEmail(),
		OrganizationID:   organizationID,
		OrganizationName: organizationName,
		PagePath:         services.NormalizeFeedbackPagePath(r.FormValue("page_path")),
	}
	if attachment != nil {
		blobKey := "support-feedback/" + uuid.NewString()
		if err := putFeedbackAttachment(r.Context(), blobKey, attachment.Content, attachment.ContentType); err != nil {
			log.Errorf("Failed to store support feedback attachment: %v", err)
			writeFeedbackError(w, http.StatusInternalServerError, "SuperPlane could not send your feedback. Try again.")
			return
		}
		message.Attachment = &messages.SupportFeedbackAttachment{
			Filename:    attachment.Filename,
			ContentType: attachment.ContentType,
			BlobKey:     blobKey,
		}
	}

	if err := publishSupportFeedback(message); err != nil {
		log.Errorf("Failed to publish support feedback: %v", err)
		if message.Attachment != nil {
			deleteFeedbackAttachment(r.Context(), message.Attachment.BlobKey)
		}
		writeFeedbackError(w, http.StatusInternalServerError, "SuperPlane could not send your feedback. Try again.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseFeedbackForm(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return r.ParseMultipartForm(int64(services.MaxSupportFeedbackRequestBytes))
	}
	return r.ParseForm()
}

func readFeedbackAttachment(r *http.Request) (*services.SupportFeedbackAttachment, error) {
	file, header, err := r.FormFile("file")
	if errors.Is(err, http.ErrMissingFile) || errors.Is(err, http.ErrNotMultipart) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	limited := io.LimitReader(file, int64(services.MaxSupportFeedbackAttachmentBytes)+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	return services.NormalizeFeedbackAttachment(header.Filename, header.Header.Get("Content-Type"), content)
}

func feedbackDetailsErrorMessage(err error) string {
	if errors.Is(err, services.ErrFeedbackDetailsTooLong) {
		return "Shorten the details and try again."
	}
	return "Enter details for this feedback."
}

func feedbackAttachmentErrorMessage(err error) string {
	switch {
	case errors.Is(err, services.ErrFeedbackAttachmentTooLarge):
		return "Attach a file that is 5 MB or smaller."
	case errors.Is(err, services.ErrFeedbackAttachmentType):
		return "Attach a PNG, JPEG, GIF, WebP, PDF, or text file."
	default:
		return "SuperPlane could not read the attached file."
	}
}

func putFeedbackAttachment(ctx context.Context, key string, content []byte, contentType string) error {
	provider := blob.Current()
	if provider == nil {
		return blob.ErrProviderNotConfigured
	}
	return provider.Put(ctx, key, bytes.NewReader(content), blob.PutOptions{ContentType: contentType})
}

func deleteFeedbackAttachment(ctx context.Context, key string) {
	provider := blob.Current()
	if provider == nil || key == "" {
		return
	}
	if err := provider.Delete(ctx, key); err != nil {
		log.Errorf("Failed to delete support feedback attachment %s: %v", key, err)
	}
}

func writeFeedbackError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
