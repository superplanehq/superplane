package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/services"
)

func TestHandleSubmitFeedback(t *testing.T) {
	email := "ada@example.com"
	user := &models.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		Name:           "Ada Lovelace",
		Email:          &email,
		Type:           models.UserTypeHuman,
	}

	t.Run("publishes a valid feedback form", func(t *testing.T) {
		var published messages.SupportFeedbackRequestedMessage
		originalPublish := publishSupportFeedback
		originalOrgName := organizationNameForFeedback
		publishSupportFeedback = func(message messages.SupportFeedbackRequestedMessage) error {
			published = message
			return nil
		}
		organizationNameForFeedback = func(organizationID string) string {
			assert.Equal(t, user.OrganizationID.String(), organizationID)
			return "Acme"
		}
		t.Cleanup(func() {
			publishSupportFeedback = originalPublish
			organizationNameForFeedback = originalOrgName
		})

		body, contentType := multipartFeedback(t, map[string]string{
			"category":  services.FeedbackCategoryBug,
			"details":   "The canvas did not load.",
			"page_path": "/acme/apps/deploy",
		}, "shot.png", "image/png", []byte("png-bytes"))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/feedback", body)
		req.Header.Set("Content-Type", contentType)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
		rec := httptest.NewRecorder()

		(&Server{}).handleSubmitFeedback(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, services.FeedbackCategoryBug, published.Category)
		assert.Equal(t, "The canvas did not load.", published.Details)
		assert.Equal(t, "Ada Lovelace", published.UserName)
		assert.Equal(t, "ada@example.com", published.UserEmail)
		assert.Equal(t, user.OrganizationID.String(), published.OrganizationID)
		assert.Equal(t, "Acme", published.OrganizationName)
		assert.Equal(t, "/acme/apps/deploy", published.PagePath)
		require.NotNil(t, published.Attachment)
		assert.Equal(t, "shot.png", published.Attachment.Filename)
		assert.Equal(t, "image/png", published.Attachment.ContentType)
		assert.Equal(t, []byte("png-bytes"), published.Attachment.Content)
	})

	t.Run("rejects a missing category", func(t *testing.T) {
		originalPublish := publishSupportFeedback
		publishSupportFeedback = func(message messages.SupportFeedbackRequestedMessage) error {
			t.Fatal("publish must not run")
			return nil
		}
		t.Cleanup(func() { publishSupportFeedback = originalPublish })

		body, contentType := multipartFeedback(t, map[string]string{
			"details": "Need help.",
		}, "", "", nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/feedback", body)
		req.Header.Set("Content-Type", contentType)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
		rec := httptest.NewRecorder()

		(&Server{}).handleSubmitFeedback(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "Select a feedback category.", jsonMessage(t, rec))
	})
}

func multipartFeedback(t *testing.T, fields map[string]string, filename, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	if filename != "" {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
		header.Set("Content-Type", contentType)
		part, err := writer.CreatePart(header)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

func jsonMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload["message"]
}
