package render

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type integrationWebhookContext struct {
	id            string
	url           string
	configuration any
	metadata      any
	secret        []byte
}

func (w *integrationWebhookContext) GetID() string              { return w.id }
func (w *integrationWebhookContext) GetURL() string             { return w.url }
func (w *integrationWebhookContext) GetSecret() ([]byte, error) { return w.secret, nil }
func (w *integrationWebhookContext) GetMetadata() any           { return w.metadata }
func (w *integrationWebhookContext) GetConfiguration() any      { return w.configuration }
func (w *integrationWebhookContext) SetSecret(secret []byte) error {
	w.secret = secret
	return nil
}

func Test__Render__Sync(t *testing.T) {
	integration := &Render{}

	t.Run("valid API key -> ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","service":{"id":"srv-1","name":"backend"}}]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","owner":{"id":"usr-123","name":"Pedro"}}]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":        "rnd_test",
				"workspacePlan": "professional",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Workspace)
		assert.Equal(t, "usr-123", metadata.Workspace.ID)
		assert.Equal(t, "professional", metadata.Workspace.Plan)

		require.Len(t, httpCtx.Requests, 2)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/services")
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/v1/owners")
	})

	t.Run("workspace not available -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","service":{"id":"srv-1","name":"backend"}}]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","owner":{"id":"usr-123","name":"Pedro"}}]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":        "rnd_test",
				"workspace":     "tea-999",
				"workspacePlan": "professional",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "is not accessible")
	})

	t.Run("organization plan -> metadata uses organization strategy", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","service":{"id":"srv-1","name":"backend"}}]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","owner":{"id":"usr-123","name":"Pedro"}}]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":        "rnd_test",
				"workspacePlan": "organization",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Workspace)
		assert.Equal(t, "usr-123", metadata.Workspace.ID)
		assert.Equal(t, "organization", metadata.Workspace.Plan)
	})

	t.Run("workspace can be selected by name", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"cursor":"x","service":{"id":"srv-1","name":"backend"}}]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"cursor":"x","owner":{"id":"usr-123","name":"Personal"}},{"cursor":"y","owner":{"id":"tea-456","name":"Acme Team"}}]`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":        "rnd_test",
				"workspace":     "Acme Team",
				"workspacePlan": "organization",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Workspace)
		assert.Equal(t, "tea-456", metadata.Workspace.ID)
		assert.Equal(t, "organization", metadata.Workspace.Plan)
	})
}

func Test__Render__ListResources(t *testing.T) {
	integration := &Render{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"cursor":"a","service":{"id":"srv-1","name":"backend"}},{"cursor":"b","service":{"id":"srv-2","name":"worker"}}]`,
				)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "rnd_test"},
		Metadata: Metadata{
			Workspace: &WorkspaceMetadata{
				ID: "usr-123",
			},
		},
	}

	resources, err := integration.ListResources("service", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "backend", resources[0].Name)
	assert.Equal(t, "srv-1", resources[0].ID)
	assert.Equal(t, "worker", resources[1].Name)
	assert.Equal(t, "srv-2", resources[1].ID)

	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "ownerId=usr-123")
}

func Test__Render__SetupWebhook(t *testing.T) {
	integration := &Render{}

	t.Run("create webhook and store secret", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"whk-1","ownerId":"usr-123","name":"SuperPlane","url":"https://hooks.superplane.dev/render","enabled":true,"eventFilter":[],"secret":"whsec-abc"}`,
					)),
				},
			},
		}

		webhookCtx := &integrationWebhookContext{
			id:            "wh_record_1",
			url:           "https://hooks.superplane.dev/render",
			configuration: struct{}{},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "rnd_test"},
			Metadata: Metadata{
				Workspace: &WorkspaceMetadata{
					ID: "usr-123",
				},
			},
		}

		metadata, err := integration.SetupWebhook(core.SetupWebhookContext{
			HTTP:        httpCtx,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "whsec-abc", string(webhookCtx.secret))

		storedMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "whk-1", storedMetadata.WebhookID)
		assert.Equal(t, "usr-123", storedMetadata.WorkspaceID)

		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/webhooks")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "ownerId=usr-123")

		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/v1/webhooks")

		body, readErr := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, readErr)

		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "usr-123", payload["ownerId"])
		assert.Equal(t, "https://hooks.superplane.dev/render", payload["url"])
		assert.Equal(t, true, payload["enabled"])
		assert.ElementsMatch(t, webhookEventFilter(WebhookConfiguration{
			Strategy: webhookStrategyIntegration,
		}), payload["eventFilter"])
	})

	t.Run("reuse existing webhook with same URL and update event filter", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"cursor":"x","webhook":{"id":"whk-existing","ownerId":"usr-123","name":"SuperPlane","url":"https://hooks.superplane.dev/render","enabled":true,"eventFilter":[],"secret":"whsec-existing"}}]`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"whk-existing","ownerId":"usr-123","name":"SuperPlane","url":"https://hooks.superplane.dev/render","enabled":true,"eventFilter":[],"secret":"whsec-existing"}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"whk-existing","ownerId":"usr-123","name":"SuperPlane","url":"https://hooks.superplane.dev/render","enabled":true,"eventFilter":["build_ended","build_started","deploy_ended","deploy_started","image_pull_failed","pipeline_minutes_exhausted","pre_deploy_ended","pre_deploy_started"],"secret":"whsec-existing"}`,
					)),
				},
			},
		}

		webhookCtx := &integrationWebhookContext{
			id:            "wh_record_1",
			url:           "https://hooks.superplane.dev/render",
			configuration: struct{}{},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "rnd_test"},
			Metadata: Metadata{
				Workspace: &WorkspaceMetadata{
					ID: "usr-123",
				},
			},
		}

		metadata, err := integration.SetupWebhook(core.SetupWebhookContext{
			HTTP:        httpCtx,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "whsec-existing", string(webhookCtx.secret))

		storedMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "whk-existing", storedMetadata.WebhookID)
		assert.Equal(t, "usr-123", storedMetadata.WorkspaceID)

		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/webhooks")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "ownerId=usr-123")
		assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/v1/webhooks/whk-existing")
		assert.Equal(t, http.MethodPatch, httpCtx.Requests[2].Method)
		assert.Contains(t, httpCtx.Requests[2].URL.String(), "/v1/webhooks/whk-existing")

		updateBody, readErr := io.ReadAll(httpCtx.Requests[2].Body)
		require.NoError(t, readErr)

		updatePayload := map[string]any{}
		require.NoError(t, json.Unmarshal(updateBody, &updatePayload))
		assert.Equal(t, "SuperPlane", updatePayload["name"])
		assert.Equal(t, "https://hooks.superplane.dev/render", updatePayload["url"])
		assert.Equal(t, true, updatePayload["enabled"])
		assert.ElementsMatch(t, webhookEventFilter(WebhookConfiguration{
			Strategy: webhookStrategyIntegration,
		}), updatePayload["eventFilter"])

	})

	t.Run("organization strategy creates resource-specific webhook when URL already exists with different filter", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"cursor":"x","webhook":{"id":"whk-existing","ownerId":"usr-123","name":"SuperPlane Deploy","url":"https://hooks.superplane.dev/render","enabled":true,"eventFilter":["deploy_ended"],"secret":"whsec-existing"}}]`,
					)),
				},
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"whk-build","ownerId":"usr-123","name":"SuperPlane Build","url":"https://hooks.superplane.dev/render","enabled":true,"eventFilter":["build_ended","build_started"],"secret":"whsec-build"}`,
					)),
				},
			},
		}

		webhookCtx := &integrationWebhookContext{
			id:  "wh_record_2",
			url: "https://hooks.superplane.dev/render",
			configuration: WebhookConfiguration{
				Strategy:     webhookStrategyResourceType,
				ResourceType: webhookResourceTypeBuild,
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "rnd_test"},
			Metadata: Metadata{
				Workspace: &WorkspaceMetadata{
					ID:   "usr-123",
					Plan: "organization",
				},
			},
		}

		metadata, err := integration.SetupWebhook(core.SetupWebhookContext{
			HTTP:        httpCtx,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "whsec-build", string(webhookCtx.secret))

		storedMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "whk-build", storedMetadata.WebhookID)

		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)

		body, readErr := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, readErr)

		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "SuperPlane Build", payload["name"])
		assert.ElementsMatch(t, webhookEventFilter(WebhookConfiguration{
			Strategy:     webhookStrategyResourceType,
			ResourceType: webhookResourceTypeBuild,
		}), payload["eventFilter"])
	})
}

func Test__Render__CleanupWebhook(t *testing.T) {
	integration := &Render{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			},
		},
	}

	err := integration.CleanupWebhook(core.CleanupWebhookContext{
		HTTP: httpCtx,
		Webhook: &integrationWebhookContext{
			metadata: WebhookMetadata{WebhookID: "whk-1", WorkspaceID: "usr-123"},
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "rnd_test"},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/webhooks/whk-1")
}

func Test__Render__CompareWebhookConfig(t *testing.T) {
	integration := &Render{}
	equal, err := integration.CompareWebhookConfig(struct{}{}, map[string]any{"eventTypes": []string{"deploy_ended"}})
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = integration.CompareWebhookConfig(
		WebhookConfiguration{
			Strategy:     webhookStrategyResourceType,
			ResourceType: webhookResourceTypeDeploy,
		},
		WebhookConfiguration{
			Strategy:     webhookStrategyResourceType,
			ResourceType: webhookResourceTypeBuild,
		},
	)
	require.NoError(t, err)
	assert.False(t, equal)

	equal, err = integration.CompareWebhookConfig(
		WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: []string{"deploy_ended"},
		},
		WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: []string{"build_ended"},
		},
	)
	require.NoError(t, err)
	assert.True(t, equal)
}

func Test__Render__MergeWebhookConfig(t *testing.T) {
	integration := &Render{}
	merged, changed, err := integration.MergeWebhookConfig(
		WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: []string{"deploy_ended"},
		},
		WebhookConfiguration{
			Strategy:   webhookStrategyIntegration,
			EventTypes: []string{"build_ended"},
		},
	)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, WebhookConfiguration{
		Strategy:   webhookStrategyIntegration,
		EventTypes: []string{"build_ended", "deploy_ended"},
	}, merged)
}
