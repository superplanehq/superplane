package dockerhub

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnImagePush__Setup(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("repository is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("valid configuration -> stores metadata and generates webhook URL", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"name":"demo","namespace":"superplane"}`)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				accessTokenSecretName: {Name: accessTokenSecretName, Value: []byte("token")},
			},
		}

		err := trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    metadata,
			Webhook:     &contexts.NodeWebhookContext{},
			Configuration: map[string]any{
				"repository": "superplane/demo",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(OnImagePushMetadata)
		require.True(t, ok)
		assert.Equal(t, "demo", stored.Repository.Name)
		assert.NotEmpty(t, stored.WebhookURL)
	})
}

func Test__OnImagePush__HandleWebhook(t *testing.T) {
	trigger := &OnImagePush{}

	t.Run("invalid JSON -> 400", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`invalid`),
			Events:        &contexts.EventContext{},
			Configuration: map[string]any{"repository": "superplane/demo"},
			Metadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{
					Repository: &RepositoryMetadata{
						Namespace: "superplane",
						Name:      "demo",
					},
				},
			},
			Logger: log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("repository mismatch -> ignored", func(t *testing.T) {
		body := []byte(`{"repository":{"name":"other", "namespace":"superplane"},"push_data":{"tag":"latest"}}`)
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Events:        events,
			Configuration: map[string]any{"repository": "superplane/demo"},
			Logger:        log.NewEntry(log.New()),
			Metadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{
					Repository: &RepositoryMetadata{
						Namespace: "superplane",
						Name:      "demo",
					},
				},
			},
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("tag filter mismatch -> ignored", func(t *testing.T) {
		body := []byte(`{"repository":{"name":"demo", "namespace":"superplane"},"push_data":{"tag":"latest"}}`)
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:   body,
			Logger: log.NewEntry(log.New()),
			Events: events,
			Metadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{
					Repository: &RepositoryMetadata{
						Namespace: "superplane",
						Name:      "demo",
					},
				},
			},
			Configuration: map[string]any{
				"repository": "superplane/demo",
				"tags": []map[string]any{
					{
						"type":  configuration.PredicateTypeEquals,
						"value": "v1.*",
					},
				},
			},
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("match -> event emitted", func(t *testing.T) {
		body := []byte(`{"repository":{"name":"demo", "namespace":"superplane"},"push_data":{"tag":"v1.2.3"}}`)
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:   body,
			Events: events,
			Logger: log.NewEntry(log.New()),
			Metadata: &contexts.MetadataContext{
				Metadata: OnImagePushMetadata{
					Repository: &RepositoryMetadata{
						Namespace: "superplane",
						Name:      "demo",
					},
				},
			},
			Configuration: map[string]any{
				"repository": "superplane/demo",
				"tags": []map[string]any{
					{
						"type":  configuration.PredicateTypeMatches,
						"value": "^v1.*",
					},
				},
			},
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "dockerhub.image.push", events.Payloads[0].Type)
	})
}
