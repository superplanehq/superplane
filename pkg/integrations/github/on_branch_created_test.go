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

func Test__OnBranchCreated__HandleWebhook(t *testing.T) {
	trigger := &OnBranchCreated{}

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
			Body:    []byte(`{"ref":"feature-branch","ref_type":"branch"}`),
			Headers: headers,
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "feature-branch"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("tag creation is ignored", func(t *testing.T) {
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
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
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

	t.Run("branch is equal -> event is emitted", func(t *testing.T) {
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
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "feature-branch"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("branch is not equal -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"develop","ref_type":"branch"}`)

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
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeNotEquals, Value: "main"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("branch matches pattern -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"feature/new-feature","ref_type":"branch"}`)

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
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: "feature/.*"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("branch does not match -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"ref":"hotfix/urgent-fix","ref_type":"branch"}`)

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
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "feature-branch"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})

	t.Run("match all branches pattern -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"any-branch-name","ref_type":"branch"}`)

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
			Configuration: OnBranchCreatedConfiguration{
				Repository: "test",
				Branches: []configuration.Predicate{
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

func Test__OnBranchCreated__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnBranchCreated{}

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
