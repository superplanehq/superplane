package circleci

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnWorkflowCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnWorkflowCompleted{}

	t.Run("no circleci-signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing signature header")
	})

	t.Run("circleci-signature without v1= prefix -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("circleci-signature", "invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  &contexts.EventContext{},
			Webhook: &contexts.WebhookContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("circleci-signature", "v1=invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"type":"workflow-completed","workflow":{"status":"success"}}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature -> event is emitted", func(t *testing.T) {
		body := []byte(`{"type":"workflow-completed","workflow":{"status":"success"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "circleci.workflow.completed", eventContext.Payloads[0].Type)
	})

	t.Run("valid signature but non-workflow event -> no event emitted", func(t *testing.T) {
		body := []byte(`{"type":"job-completed","job":{"status":"success"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte(`invalid json`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})
}

func Test__OnWorkflowCompleted__Setup(t *testing.T) {
	trigger := OnWorkflowCompleted{}

	t.Run("projectSlug is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnWorkflowCompletedConfiguration{ProjectSlug: ""},
		})

		require.ErrorContains(t, err, "projectSlug is required")
	})

	t.Run("metadata already set -> returns early", func(t *testing.T) {
		testProject := &ProjectInfo{ID: "proj-123", Name: "test-project", Slug: "gh/myorg/test-project", URL: "https://app.circleci.com/pipelines/gh/myorg/test-project"}

		metadataCtx := &contexts.MetadataContext{
			Metadata: OnWorkflowCompletedMetadata{
				Project: testProject,
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Configuration: OnWorkflowCompletedConfiguration{ProjectSlug: "gh/myorg/test-project"},
		})

		require.NoError(t, err)
		metadata := metadataCtx.Get().(OnWorkflowCompletedMetadata)
		assert.Equal(t, testProject, metadata.Project)
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}
