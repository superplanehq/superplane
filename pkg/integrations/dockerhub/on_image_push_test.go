package dockerhub

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnImagePush__Setup(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("missing namespace -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"namespace":  "",
				"repository": "myapp",
			},
			Metadata: &contexts.MetadataContext{},
			HTTP:     &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("missing repository -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "",
			},
			Metadata: &contexts.MetadataContext{},
			HTTP:     &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "testuser",
				"accessToken": "test-token",
			},
		}
		webhookCtx := &contexts.WebhookContext{
			Secret: "test-secret",
		}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Login response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"token": "test-jwt-token"}`)),
				},
				// Get repository response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"namespace": "myorg",
						"name": "myapp",
						"description": "My app"
					}`)),
				},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
			},
			Metadata:    metadata,
			Integration: integration,
			Webhook:     webhookCtx,
			HTTP:        httpContext,
		})

		require.NoError(t, err)

		// Verify metadata was set
		m := metadata.Get().(OnImagePushMetadata)
		assert.NotNil(t, m.Repository)
		assert.Equal(t, "myorg", m.Repository.Namespace)
		assert.Equal(t, "myapp", m.Repository.Name)
		assert.Equal(t, "myorg/myapp", m.Repository.FullName)
		assert.NotEmpty(t, m.WebhookURL)
	})

	t.Run("already setup -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnImagePushMetadata{
				Repository: &RepositoryMetadata{
					Namespace: "myorg",
					Name:      "myapp",
					FullName:  "myorg/myapp",
				},
				WebhookURL: "https://example.com/webhooks/123",
			},
		}
		integration := &contexts.IntegrationContext{}
		webhookCtx := &contexts.WebhookContext{}
		httpContext := &contexts.HTTPContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
			},
			Metadata:    metadata,
			Integration: integration,
			Webhook:     webhookCtx,
			HTTP:        httpContext,
		})

		require.NoError(t, err)

		// No HTTP requests should be made
		assert.Len(t, httpContext.Requests, 0)
	})
}

func Test__OnImagePush__HandleWebhook(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("invalid JSON -> returns bad request", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte("not json"),
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
			},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.Error(t, err)
	})

	t.Run("different repository -> returns OK without event", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"push_data": {"tag": "latest", "pusher": "testuser"},
				"repository": {"repo_name": "other/repo", "namespace": "other"}
			}`),
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
			},
			Events: events,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Len(t, events.Payloads, 0)
	})

	t.Run("matching repository -> emits event", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"push_data": {"tag": "latest", "pusher": "testuser"},
				"repository": {"repo_name": "myorg/myapp", "namespace": "myorg", "name": "myapp"}
			}`),
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
			},
			Events: events,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, "dockerhub.imagePush", events.Payloads[0].Type)
	})

	t.Run("tag filter matches -> emits event", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"push_data": {"tag": "v1.2.3", "pusher": "testuser"},
				"repository": {"repo_name": "myorg/myapp", "namespace": "myorg"}
			}`),
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
				"tags": []map[string]any{
					{"type": configuration.PredicateTypeMatches, "value": "v.*"},
				},
			},
			Events: events,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("tag filter does not match -> no event", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"push_data": {"tag": "latest", "pusher": "testuser"},
				"repository": {"repo_name": "myorg/myapp", "namespace": "myorg"}
			}`),
			Configuration: map[string]any{
				"namespace":  "myorg",
				"repository": "myapp",
				"tags": []map[string]any{
					{"type": configuration.PredicateTypeMatches, "value": "v.*"},
				},
			},
			Events: events,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Len(t, events.Payloads, 0)
	})
}
