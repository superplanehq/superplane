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

func Test__OnPush__HandleWebhook(t *testing.T) {
	trigger := &OnPush{}

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:        headers,
			EventContext:   &contexts.EventContext{},
			WebhookContext: &contexts.WebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "push")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"ref":"refs/heads/main"}`),
			Headers: headers,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "test"},
				Refs: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			WebhookContext: &contexts.WebhookContext{Secret: secret},
			EventContext:   &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("branch deletion push is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main","deleted":true}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "test"},
				Refs: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			WebhookContext: &contexts.WebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("ref is equal -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "test"},
				Refs: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			WebhookContext: &contexts.WebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("ref is not equal -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/feat/1"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "test"},
				Refs: []configuration.Predicate{
					{Type: configuration.PredicateTypeNotEquals, Value: "refs/heads/main"},
				},
			},
			WebhookContext: &contexts.WebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("ref matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/feat/1"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "test"},
				Refs: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: "refs/heads/feat/*"},
				},
			},
			WebhookContext: &contexts.WebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("ref is not equal -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/patch-1"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "test"},
				Refs: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			WebhookContext: &contexts.WebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})
}

func Test__OnPush__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnPush{}

	t.Run("repository is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := trigger.Setup(core.TriggerContext{
			AppInstallationContext: appCtx,
			MetadataContext:        &contexts.MetadataContext{},
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: ""},
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			AppInstallationContext: appCtx,
			MetadataContext:        &contexts.MetadataContext{},
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "world"},
			},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set and webhook is requested", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			AppInstallationContext: appCtx,
			MetadataContext:        &nodeMetadataCtx,
			Configuration: OnPushConfiguration{
				BaseRepositoryConfig: BaseRepositoryConfig{Repository: "hello"},
			},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		require.Len(t, appCtx.WebhookRequests, 1)

		webhookRequest := appCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, webhookRequest.EventType, "push")
		assert.Equal(t, webhookRequest.Repository, "hello")
	})
}

func Test__IsBranchDeletionEvent(t *testing.T) {
	assert.True(t, isBranchDeletionEvent(map[string]any{"deleted": true}))
	assert.False(t, isBranchDeletionEvent(map[string]any{"deleted": false}))
	assert.False(t, isBranchDeletionEvent(map[string]any{}))
	assert.False(t, isBranchDeletionEvent(map[string]any{}))
}
