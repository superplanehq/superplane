package semaphore

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPipelineDone__HandleWebhook(t *testing.T) {
	trigger := &OnPipelineDone{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("no X-Semaphore-Signature-256 -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Logger:  logger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("X-Semaphore-Signature-256 without sha256= prefix -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Semaphore-Signature-256", "invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  &contexts.EventContext{},
			Webhook: &contexts.WebhookContext{},
			Logger:  logger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Semaphore-Signature-256", "sha256=invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"pipeline":{"state":"done"}}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
			Logger:  logger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature -> event is emitted", func(t *testing.T) {
		secret := "test-secret"
		body := []byte(`{"revision":{"reference":"refs/heads/main"},"pipeline":{"state":"done","result":"passed","yaml_file_name":"semaphore.yml"}}`)
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "semaphore.pipeline.done", eventContext.Payloads[0].Type)
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte(`invalid json`)

		secret := "test-secret"
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("ref filter match -> event is emitted", func(t *testing.T) {
		body := []byte(`{"revision":{"reference":"refs/heads/main"},"pipeline":{"result":"passed","yaml_file_name":"semaphore.yml"}}`)
		secret := "test-secret"
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("ref filter mismatch -> event is ignored", func(t *testing.T) {
		body := []byte(`{"revision":{"reference":"refs/heads/feature"},"pipeline":{"result":"passed","yaml_file_name":"semaphore.yml"}}`)
		secret := "test-secret"
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("results filter mismatch -> event is ignored", func(t *testing.T) {
		body := []byte(`{"revision":{"reference":"refs/heads/main"},"pipeline":{"result":"failed","yaml_file_name":"semaphore.yml"}}`)
		secret := "test-secret"
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"results": []string{"passed"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("pipeline filter match -> event is emitted", func(t *testing.T) {
		body := []byte(`{"revision":{"reference":"refs/heads/main"},"pipeline":{"result":"passed","working_directory":".semaphore","yaml_file_name":"production.yml"}}`)
		secret := "test-secret"
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"pipelines": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: ".semaphore/production.yml"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("missing pipeline result with results filter -> 400", func(t *testing.T) {
		body := []byte(`{"pipeline":{"yaml_file_name":"semaphore.yml"}}`)
		secret := "test-secret"
		headers := buildSemaphoreHeaders(secret, body)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"results": []string{"passed"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Logger:  logger,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing pipeline.result")
		assert.Zero(t, eventContext.Count())
	})
}

func Test__OnPipelineDone__Setup(t *testing.T) {
	trigger := OnPipelineDone{}

	t.Run("project is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: OnPipelineDoneConfiguration{Project: ""},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("metadata already set -> returns early", func(t *testing.T) {
		testProject := &Project{ID: "proj-123", Name: "test-project", URL: "https://example.semaphoreci.com/projects/proj-123"}

		metadataCtx := &contexts.MetadataContext{
			Metadata: OnPipelineDoneMetadata{
				Project: testProject,
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Configuration: OnPipelineDoneConfiguration{Project: "test-project"},
		})

		require.NoError(t, err)
		metadata := metadataCtx.Get().(OnPipelineDoneMetadata)
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

func buildSemaphoreHeaders(secret string, body []byte) http.Header {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	signature := fmt.Sprintf("%x", h.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Semaphore-Signature-256", "sha256="+signature)

	return headers
}
