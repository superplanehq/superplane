package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnTagCreated__HandleWebhook(t *testing.T) {
	trigger := &OnTagCreated{}

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "create")
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  &contexts.EventContext{},
			Webhook: &contexts.NodeWebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "create")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"ref":"v1.0.0","ref_type":"tag"}`),
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "v1.0.0"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("branch creation is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"feature-branch","ref_type":"branch"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "create")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: ".*"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("tag is equal -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"v1.0.0","ref_type":"tag"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "create")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "v1.0.0"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("tag is not equal -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"v2.0.0","ref_type":"tag"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "create")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeNotEquals, Value: "v1.0.0"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("tag matches pattern -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"v1.2.3","ref_type":"tag"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "create")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: "v.*"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("tag does not match -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"ref":"release-2024","ref_type":"tag"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "create")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "v1.0.0"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})

	t.Run("match all tags pattern -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"some-random-tag","ref_type":"tag"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "create")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnTagCreatedConfiguration{
				Repository: "test",
				Tags: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: ".*"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})
}

func Test__OnTagCreated__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnTagCreated{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set and webhook is requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookRequest := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, webhookRequest.EventType, "create")
		assert.Equal(t, webhookRequest.Repository, "hello")
	})
}
