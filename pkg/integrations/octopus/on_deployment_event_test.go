package octopus

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Octopus_OnDeploymentEvent__Setup(t *testing.T) {
	trigger := &OnDeploymentEvent{}

	t.Run("requests webhook with selected categories", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentSucceeded", "DeploymentFailed"},
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"DeploymentFailed", "DeploymentSucceeded"}, webhookConfig.EventCategories)
		assert.Empty(t, webhookConfig.Projects)
		assert.Empty(t, webhookConfig.Environments)
	})

	t.Run("defaults to succeeded and failed when no categories specified", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metadataCtx,
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, deploymentDefaultEventCategories, webhookConfig.EventCategories)
	})

	t.Run("includes project and environment filters", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentSucceeded"},
				"project":         "Projects-1",
				"environment":     "Environments-2",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"DeploymentSucceeded"}, webhookConfig.EventCategories)
		assert.Equal(t, []string{"Projects-1"}, webhookConfig.Projects)
		assert.Equal(t, []string{"Environments-2"}, webhookConfig.Environments)
	})
}

func Test__Octopus_OnDeploymentEvent__HandleWebhook(t *testing.T) {
	trigger := &OnDeploymentEvent{}

	payload := map[string]any{
		"Timestamp": "2026-01-15T10:30:00.000Z",
		"EventType": "SubscriptionPayload",
		"Payload": map[string]any{
			"ServerUri": "https://octopus.example.com",
			"Event": map[string]any{
				"Category": "DeploymentSucceeded",
				"Message":  "Deploy succeeded for project MyProject",
				"Occurred": "2026-01-15T10:29:55.000Z",
				"RelatedDocumentIds": []any{
					"Projects-1",
					"Environments-2",
					"Deployments-100",
					"Releases-50",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "test-webhook-secret"

	t.Run("missing webhook header -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{"Content-Type": []string{"application/json"}},
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "missing X-SuperPlane-Webhook-Secret header")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("invalid secret -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{"wrong-secret"},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "invalid webhook secret")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("non-JSON content type -> 200, no events", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"text/plain"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("empty event category -> 200, no events", func(t *testing.T) {
		emptyPayload := map[string]any{
			"Timestamp": "2026-01-15T10:30:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category": "",
				},
			},
		}
		emptyBody, marshalErr := json.Marshal(emptyPayload)
		require.NoError(t, marshalErr)

		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          emptyBody,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("non-deployment event category -> 200, no events", func(t *testing.T) {
		nonDeployPayload := map[string]any{
			"Timestamp": "2026-01-15T10:30:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category": "MachineHealthChanged",
				},
			},
		}
		nonDeployBody, marshalErr := json.Marshal(nonDeployPayload)
		require.NoError(t, marshalErr)

		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          nonDeployBody,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("category not in filter -> 200, no events", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentFailed"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("project filter mismatch -> 200, no events", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentSucceeded"},
				"project":         "Projects-999",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("environment filter mismatch -> 200, no events", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentSucceeded"},
				"environment":     "Environments-999",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("valid event with matching filters -> event emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentSucceeded"},
				"project":         "Projects-1",
				"environment":     "Environments-2",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "octopus.deployment.succeeded", eventCtx.Payloads[0].Type)

		data, ok := eventCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "DeploymentSucceeded", data["eventType"])
		assert.Equal(t, "DeploymentSucceeded", data["category"])
		assert.Equal(t, "2026-01-15T10:30:00.000Z", data["timestamp"])
		assert.Equal(t, "Projects-1", data["projectId"])
		assert.Equal(t, "Environments-2", data["environmentId"])
		assert.Equal(t, "Deployments-100", data["deploymentId"])
		assert.Equal(t, "Releases-50", data["releaseId"])
		assert.Equal(t, "https://octopus.example.com", data["serverUri"])
		assert.Equal(t, "Deploy succeeded for project MyProject", data["message"])
	})

	t.Run("valid event with default categories -> event emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "octopus.deployment.succeeded", eventCtx.Payloads[0].Type)
	})

	t.Run("DeploymentQueued event -> correct payload type", func(t *testing.T) {
		queuedPayload := map[string]any{
			"Timestamp": "2026-01-15T10:30:00.000Z",
			"EventType": "SubscriptionPayload",
			"Payload": map[string]any{
				"Event": map[string]any{
					"Category":           "DeploymentQueued",
					"RelatedDocumentIds": []any{"Projects-1"},
				},
			},
		}
		queuedBody, marshalErr := json.Marshal(queuedPayload)
		require.NoError(t, marshalErr)

		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    queuedBody,
			Headers: headers,
			Configuration: map[string]any{
				"eventCategories": []string{"DeploymentQueued"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "octopus.deployment.queued", eventCtx.Payloads[0].Type)
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := http.Header{
			"Content-Type":                []string{"application/json"},
			"X-Superplane-Webhook-Secret": []string{secret},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not json"),
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusBadRequest, status)
		assert.ErrorContains(t, webhookErr, "error parsing request body")
		assert.Zero(t, eventCtx.Count())
	})
}
