package railway

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Railway__OnDeployment__Setup(t *testing.T) {
	trigger := &OnDeploymentEvent{}

	t.Run("success", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project":    "p-1",
				"eventTypes": []string{"Deployment.deployed"},
			},
			Integration: intCtx,
		})
		require.NoError(t, err)
		require.Len(t, intCtx.WebhookRequests, 1)

		config, ok := intCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "p-1", config.ProjectID)
		assert.Equal(t, []string{"Deployment.deployed"}, config.EventTypes)
	})

	t.Run("validation failure", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project": "",
			},
			Integration: intCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})
}

func Test__Railway__OnDeployment__HandleWebhook(t *testing.T) {
	trigger := &OnDeploymentEvent{}

	t.Run("emits matching payload", func(t *testing.T) {
		payload := OnDeploymentPayload{
			Type:         "Deployment.deployed",
			ProjectID:    "p-1",
			DeploymentID: "deploy-123",
		}
		body, _ := json.Marshal(payload)

		eventCtx := &contexts.EventContext{}
		code, res, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"project":    "p-1",
				"eventTypes": []string{"Deployment.deployed"},
			},
			Body:   body,
			Events: eventCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Nil(t, res)

		require.Len(t, eventCtx.Payloads, 1)
		assert.Equal(t, "railway.onDeployment", eventCtx.Payloads[0].Type)

		emittedPayload, ok := eventCtx.Payloads[0].Data.([]any)
		require.True(t, ok)
		require.Len(t, emittedPayload, 1)

		var parsed OnDeploymentPayload
		err = mapstructureDecode(emittedPayload[0], &parsed)
		require.NoError(t, err)
		assert.Equal(t, "Deployment.deployed", parsed.Type)
		assert.Equal(t, "p-1", parsed.ProjectID)
	})

	t.Run("ignores mismatched projectID", func(t *testing.T) {
		payload := OnDeploymentPayload{
			Type:         "Deployment.deployed",
			ProjectID:    "other-project",
			DeploymentID: "deploy-123",
		}
		body, _ := json.Marshal(payload)

		eventCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"project":    "p-1",
				"eventTypes": []string{"Deployment.deployed"},
			},
			Body:   body,
			Events: eventCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Len(t, eventCtx.Payloads, 0)
	})

	t.Run("emits Railway nested resource payload", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.deployed",
			"details": { "status": "SUCCESS" },
			"resource": {
				"project": { "id": "p-1" },
				"environment": { "id": "e-1" },
				"service": { "id": "s-1" },
				"deployment": { "id": "deploy-123" }
			},
			"timestamp": "2025-11-21T23:48:42.311Z"
		}`)

		eventCtx := &contexts.EventContext{}
		code, res, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"project":    "p-1",
				"eventTypes": []string{"Deployment.deployed"},
			},
			Body:   body,
			Events: eventCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Nil(t, res)
		require.Len(t, eventCtx.Payloads, 1)

		emittedPayload, ok := eventCtx.Payloads[0].Data.([]any)
		require.True(t, ok)
		var parsed OnDeploymentPayload
		err = mapstructureDecode(emittedPayload[0], &parsed)
		require.NoError(t, err)
		assert.Equal(t, "p-1", parsed.ProjectID)
		assert.Equal(t, "deploy-123", parsed.DeploymentID)
		assert.Equal(t, "SUCCESS", parsed.Status)
	})

	t.Run("ignores mismatched eventType", func(t *testing.T) {
		payload := OnDeploymentPayload{
			Type:         "Deployment.failed",
			ProjectID:    "p-1",
			DeploymentID: "deploy-123",
		}
		body, _ := json.Marshal(payload)

		eventCtx := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Configuration: map[string]any{
				"project":    "p-1",
				"eventTypes": []string{"Deployment.deployed"},
			},
			Body:   body,
			Events: eventCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Len(t, eventCtx.Payloads, 0)
	})
}

func mapstructureDecode(input any, output any) error {
	var config = &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  output,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
