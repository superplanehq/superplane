package harness

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type webhookContextMock struct {
	id            string
	url           string
	secret        string
	metadata      any
	configuration any
}

func (w *webhookContextMock) GetID() string {
	return w.id
}

func (w *webhookContextMock) GetURL() string {
	return w.url
}

func (w *webhookContextMock) GetSecret() ([]byte, error) {
	return []byte(w.secret), nil
}

func (w *webhookContextMock) GetMetadata() any {
	return w.metadata
}

func (w *webhookContextMock) GetConfiguration() any {
	return w.configuration
}

func (w *webhookContextMock) SetSecret(secret []byte) error {
	w.secret = string(secret)
	return nil
}

func Test__HarnessWebhookHandler__CompareConfig(t *testing.T) {
	handler := &HarnessWebhookHandler{}

	equal, err := handler.CompareConfig(
		WebhookConfiguration{
			PipelineIdentifier: "deploy",
			OrgID:              "default",
			ProjectID:          "default_project",
			EventTypes:         []string{"PipelineEnd"},
		},
		WebhookConfiguration{
			PipelineIdentifier: "deploy",
			OrgID:              "default",
			ProjectID:          "default_project",
			EventTypes:         []string{"pipeline_end"},
		},
	)
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = handler.CompareConfig(
		WebhookConfiguration{
			PipelineIdentifier: "deploy",
			OrgID:              "default",
			ProjectID:          "default_project",
			EventTypes:         []string{"PipelineEnd"},
		},
		WebhookConfiguration{
			PipelineIdentifier: "release",
			OrgID:              "default",
			ProjectID:          "default_project",
			EventTypes:         []string{"PipelineEnd"},
		},
	)
	require.NoError(t, err)
	assert.False(t, equal)
}

func Test__HarnessWebhookHandler__Merge(t *testing.T) {
	handler := &HarnessWebhookHandler{}

	t.Run("unchanged config does not force reprovision", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{
				PipelineIdentifier: "deploy",
				OrgID:              "default",
				ProjectID:          "default_project",
				EventTypes:         []string{"PipelineEnd"},
			},
			WebhookConfiguration{
				PipelineIdentifier: "deploy",
				OrgID:              "default",
				ProjectID:          "default_project",
				EventTypes:         []string{"pipeline_end"},
			},
		)
		require.NoError(t, err)
		assert.False(t, changed)

		config := WebhookConfiguration{}
		require.NoError(t, mapstructure.Decode(merged, &config))
		assert.Equal(t, "deploy", config.PipelineIdentifier)
		assert.Equal(t, "default", config.OrgID)
		assert.Equal(t, "default_project", config.ProjectID)
		assert.Equal(t, []string{"PipelineEnd"}, normalizeWebhookEventTypes(config.EventTypes))
	})

	t.Run("changed config triggers reprovision", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{
				PipelineIdentifier: "deploy",
				OrgID:              "default",
				ProjectID:          "default_project",
				EventTypes:         []string{"PipelineEnd"},
			},
			WebhookConfiguration{
				PipelineIdentifier: "release",
				OrgID:              "default",
				ProjectID:          "default_project",
				EventTypes:         []string{"PipelineEnd"},
			},
		)
		require.NoError(t, err)
		assert.True(t, changed)

		config := WebhookConfiguration{}
		require.NoError(t, mapstructure.Decode(merged, &config))
		assert.Equal(t, "release", config.PipelineIdentifier)
	})
}

func Test__HarnessWebhookHandler__SetupAndCleanup(t *testing.T) {
	handler := &HarnessWebhookHandler{}
	webhookID := "webhook-123"
	hash := sha256.Sum256([]byte(webhookID))
	base := fmt.Sprintf("superplane-harness-%x", hash[:8])
	expectedRuleIdentifier := normalizeHarnessIdentifier(base + "-rule")

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"data":{"yamlPipeline":"pipeline:\n  identifier: Superplane_Test\n  notificationRules: []\n"}}`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"SUCCESS"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(fmt.Sprintf(
					`{"data":{"yamlPipeline":"pipeline:\n  identifier: Superplane_Test\n  notificationRules:\n    - identifier: %s\n      name: %s\n      enabled: true\n      pipelineEvents:\n        - type: PipelineEnd\n      notificationMethod:\n        type: Webhook\n        spec:\n          webhookUrl: https://example.com/api/v1/webhooks/webhook-123\n"}}`,
					expectedRuleIdentifier,
					expectedRuleIdentifier,
				))),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"SUCCESS"}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	}

	webhookCtx := &webhookContextMock{
		id:     webhookID,
		url:    "https://example.com/api/v1/webhooks/webhook-123",
		secret: "secret-value",
		configuration: WebhookConfiguration{
			PipelineIdentifier: "Superplane_Test",
			OrgID:              "default",
			ProjectID:          "default_project",
			EventTypes:         []string{"PipelineEnd"},
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)

	decoded := WebhookMetadata{}
	require.NoError(t, mapstructure.Decode(metadata, &decoded))
	assert.Equal(t, "Superplane_Test", decoded.PipelineIdentifier)
	assert.Equal(t, "default", decoded.OrgID)
	assert.Equal(t, "default_project", decoded.ProjectID)
	assert.Equal(t, expectedRuleIdentifier, decoded.RuleIdentifier)
	assert.Equal(t, "https://example.com/api/v1/webhooks/webhook-123", decoded.URL)

	require.Len(t, httpCtx.Requests, 2)
	setupPayload, readErr := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(setupPayload), expectedRuleIdentifier)
	assert.Contains(t, string(setupPayload), "https://example.com/api/v1/webhooks/webhook-123")
	assert.Contains(t, string(setupPayload), "Authorization")
	assert.Contains(t, string(setupPayload), "Bearer secret-value")

	webhookCtx.metadata = decoded
	err = handler.Cleanup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 4)
}

func Test__HarnessWebhookHandler__Setup_WithoutPipeline_FallsBackToPolling(t *testing.T) {
	handler := &HarnessWebhookHandler{}
	httpCtx := &contexts.HTTPContext{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	}

	webhookCtx := &webhookContextMock{
		id:            "webhook-123",
		url:           "https://example.com/api/v1/webhooks/webhook-123",
		secret:        "secret-value",
		configuration: WebhookConfiguration{EventTypes: []string{"PipelineEnd"}},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)

	decoded := WebhookMetadata{}
	require.NoError(t, mapstructure.Decode(metadata, &decoded))
	assert.Empty(t, decoded.RuleIdentifier)
	assert.Equal(t, "https://example.com/api/v1/webhooks/webhook-123", decoded.URL)
	assert.Len(t, httpCtx.Requests, 0)
}

func Test__HarnessWebhookHandler__Setup_ReturnsErrorWhenProvisioningFails(t *testing.T) {
	handler := &HarnessWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"data":{"yamlPipeline":"pipeline:\n  identifier: Superplane_Test\n  notificationRules: []\n"}}`,
				)),
			},
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"message":"Oops, something went wrong on our end."}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	}

	webhookCtx := &webhookContextMock{
		id:     "webhook-123",
		url:    "https://example.com/api/v1/webhooks/webhook-123",
		secret: "secret-value",
		configuration: WebhookConfiguration{
			PipelineIdentifier: "Superplane_Test",
			OrgID:              "default",
			ProjectID:          "default_project",
			EventTypes:         []string{"PipelineEnd"},
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.ErrorContains(t, err, "failed to provision Harness notification resources")
	assert.Nil(t, metadata)
}
