package public

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/services"
)

var feedbackPNG = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

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
		originalOrg := resolveFeedbackOrganization
		publishSupportFeedback = func(message messages.SupportFeedbackRequestedMessage) error {
			published = message
			return nil
		}
		requestedOrg := uuid.NewString()
		resolveFeedbackOrganization = func(r *http.Request, current *models.User) (string, string, error) {
			assert.Equal(t, requestedOrg, r.Header.Get("x-organization-id"))
			assert.NotEqual(t, current.OrganizationID.String(), requestedOrg)
			return requestedOrg, "Acme", nil
		}
		store, err := filesystem.New(t.TempDir())
		require.NoError(t, err)
		blob.SetCurrent(store)
		t.Cleanup(func() {
			publishSupportFeedback = originalPublish
			resolveFeedbackOrganization = originalOrg
			blob.SetCurrent(nil)
		})

		body, contentType := multipartFeedback(t, map[string]string{
			"category":  services.FeedbackCategoryBug,
			"details":   "The canvas did not load.",
			"page_path": "/acme/apps/deploy",
		}, "shot.png", "image/png", feedbackPNG)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/feedback", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("x-organization-id", requestedOrg)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
		rec := httptest.NewRecorder()

		(&Server{}).handleSubmitFeedback(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, services.FeedbackCategoryBug, published.Category)
		assert.Equal(t, "The canvas did not load.", published.Details)
		assert.Equal(t, "Ada Lovelace", published.UserName)
		assert.Equal(t, "ada@example.com", published.UserEmail)
		assert.Equal(t, requestedOrg, published.OrganizationID)
		assert.Equal(t, "Acme", published.OrganizationName)
		assert.Equal(t, "/acme/apps/deploy", published.PagePath)
		require.NotNil(t, published.Attachment)
		assert.Equal(t, "shot.png", published.Attachment.Filename)
		assert.Equal(t, "image/png", published.Attachment.ContentType)
		assert.NotEmpty(t, published.Attachment.BlobKey)
		reader, err := store.Get(context.Background(), published.Attachment.BlobKey)
		require.NoError(t, err)
		defer reader.Close()
		stored, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, feedbackPNG, stored)
	})

	t.Run("rejects feedback for an organization the user does not belong to", func(t *testing.T) {
		originalPublish := publishSupportFeedback
		originalOrg := resolveFeedbackOrganization
		publishSupportFeedback = func(message messages.SupportFeedbackRequestedMessage) error {
			t.Fatal("publish must not run")
			return nil
		}
		resolveFeedbackOrganization = func(r *http.Request, _ *models.User) (string, string, error) {
			return "", "", errFeedbackOrganization
		}
		t.Cleanup(func() {
			publishSupportFeedback = originalPublish
			resolveFeedbackOrganization = originalOrg
		})

		body, contentType := multipartFeedback(t, map[string]string{
			"category": services.FeedbackCategoryBug,
			"details":  "The canvas did not load.",
		}, "", "", nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/feedback", body)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("x-organization-id", uuid.NewString())
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
		rec := httptest.NewRecorder()

		(&Server{}).handleSubmitFeedback(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("rejects a request that exceeds the body limit", func(t *testing.T) {
		originalPublish := publishSupportFeedback
		publishSupportFeedback = func(message messages.SupportFeedbackRequestedMessage) error {
			t.Fatal("publish must not run")
			return nil
		}
		t.Cleanup(func() { publishSupportFeedback = originalPublish })

		body, contentType := multipartFeedback(t, map[string]string{
			"category": services.FeedbackCategoryBug,
			"details":  "The canvas did not load.",
		}, "huge.bin", "application/octet-stream", bytes.Repeat([]byte("a"), services.MaxSupportFeedbackRequestBytes+1))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/me/feedback", body)
		req.Header.Set("Content-Type", contentType)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
		rec := httptest.NewRecorder()

		(&Server{}).handleSubmitFeedback(rec, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Equal(t, "The feedback form is too large.", jsonMessage(t, rec))
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
