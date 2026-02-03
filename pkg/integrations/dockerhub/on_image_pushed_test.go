package dockerhub

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnImagePushed__Setup(t *testing.T) {
	trigger := &OnImagePushed{}

	t.Run("missing repository -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"repository": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"repository": "myorg/myapp",
			},
			Metadata:    metadata,
			Integration: integration,
		})

		require.NoError(t, err)

		// Verify metadata was set
		m := metadata.Get().(OnImagePushedMetadata)
		assert.Equal(t, "myorg/myapp", m.Repository)

		// Verify webhook was requested
		require.Len(t, integration.WebhookRequests, 1)
		webhookConfig := integration.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "myorg/myapp", webhookConfig.Repository)
	})

	t.Run("already setup -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnImagePushedMetadata{
				Repository: "myorg/myapp",
			},
		}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"repository": "myorg/myapp",
			},
			Metadata:    metadata,
			Integration: integration,
		})

		require.NoError(t, err)

		// No new webhook request should be made
		assert.Len(t, integration.WebhookRequests, 0)
	})
}

func Test__OnImagePushed__HandleWebhook(t *testing.T) {
	trigger := &OnImagePushed{}

	t.Run("valid push event -> emits event", func(t *testing.T) {
		body := []byte(`{
			"callback_url": "https://registry.hub.docker.com/...",
			"push_data": {
				"pushed_at": 1706803200,
				"pusher": "johndoe",
				"tag": "v1.2.3"
			},
			"repository": {
				"repo_name": "myorg/myapp",
				"name": "myapp",
				"namespace": "myorg"
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"repository": "myorg/myapp",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())

		// Verify emitted event
		payload := eventContext.Payloads[0]
		assert.Equal(t, "dockerhub.imagePushed", payload.Type)
	})

	t.Run("different repository -> no event emitted", func(t *testing.T) {
		body := []byte(`{
			"push_data": {
				"tag": "latest"
			},
			"repository": {
				"repo_name": "other/repo"
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"repository": "myorg/myapp",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("tag filter matches -> emits event", func(t *testing.T) {
		body := []byte(`{
			"push_data": {
				"tag": "v1.2.3"
			},
			"repository": {
				"repo_name": "myorg/myapp"
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"repository": "myorg/myapp",
				"tagFilter":  "v*",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("tag filter does not match -> no event", func(t *testing.T) {
		body := []byte(`{
			"push_data": {
				"tag": "latest"
			},
			"repository": {
				"repo_name": "myorg/myapp"
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"repository": "myorg/myapp",
				"tagFilter":  "v*",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("invalid JSON body -> error", func(t *testing.T) {
		body := []byte(`invalid json`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: body,
			Configuration: map[string]any{
				"repository": "myorg/myapp",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.Error(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})
}

func Test__matchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		pattern  string
		expected bool
	}{
		{"empty pattern matches all", "anything", "", true},
		{"exact match", "latest", "latest", true},
		{"exact mismatch", "latest", "main", false},
		{"prefix wildcard match", "v1.2.3", "v*", true},
		{"prefix wildcard mismatch", "release-1.0", "v*", false},
		{"suffix wildcard match", "feature-latest", "*-latest", true},
		{"suffix wildcard mismatch", "feature-main", "*-latest", false},
		{"single wildcard matches anything", "anything", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.s, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}
