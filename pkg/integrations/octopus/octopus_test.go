package octopus

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

// integrationWebhookContext implements core.IntegrationWebhookContext for testing
// the OctopusWebhookHandler Setup/Cleanup flow.
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

func Test__Octopus__Sync(t *testing.T) {
	integration := &Octopus{}

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ValidateCredentials (GET /api/users/me)
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"Id":"Users-1","Username":"admin"}`)),
				},
				// ListSpaces (GET /api/spaces/all)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
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
		require.NotNil(t, metadata.Space)
		assert.Equal(t, "Spaces-1", metadata.Space.ID)
		assert.Equal(t, "Default", metadata.Space.Name)

		require.Len(t, httpCtx.Requests, 2)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/users/me")
		assert.Contains(t, httpCtx.Requests[1].URL.Path, "/api/spaces/all")
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"ErrorMessage":"Unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "invalid-key",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "failed to verify Octopus Deploy credentials")
	})

	t.Run("missing serverUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "API-TEST",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "serverUrl is required")
	})

	t.Run("missing apiKey -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("specific space selected -> resolved", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ValidateCredentials
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"Id":"Users-1","Username":"admin"}`)),
				},
				// ListSpaces
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true},{"Id":"Spaces-2","Name":"Production","IsDefault":false}]`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
				"space":     "Production",
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
		require.NotNil(t, metadata.Space)
		assert.Equal(t, "Spaces-2", metadata.Space.ID)
		assert.Equal(t, "Production", metadata.Space.Name)
	})

	t.Run("space not found -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ValidateCredentials
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"Id":"Users-1","Username":"admin"}`)),
				},
				// ListSpaces
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
				"space":     "NonExistent",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "is not accessible")
	})
}

func Test__Octopus__ListResources(t *testing.T) {
	integration := &Octopus{}

	t.Run("list projects", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListProjects
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Projects-1","Name":"Backend API"},{"Id":"Projects-2","Name":"Frontend App"}]`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
			Metadata: Metadata{
				Space: &SpaceMetadata{ID: "Spaces-1", Name: "Default"},
			},
		}

		resources, err := integration.ListResources("project", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "Backend API", resources[0].Name)
		assert.Equal(t, "Projects-1", resources[0].ID)
		assert.Equal(t, "Frontend App", resources[1].Name)
		assert.Equal(t, "Projects-2", resources[1].ID)

		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/Spaces-1/projects/all")
	})

	t.Run("list environments", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Environments-1","Name":"Development"},{"Id":"Environments-2","Name":"Production"}]`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
			Metadata: Metadata{
				Space: &SpaceMetadata{ID: "Spaces-1", Name: "Default"},
			},
		}

		resources, err := integration.ListResources("environment", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "Development", resources[0].Name)
		assert.Equal(t, "Production", resources[1].Name)
	})

	t.Run("list releases with project parameter", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Items":[{"Id":"Releases-1","Version":"1.0.0","ProjectId":"Projects-1"},{"Id":"Releases-2","Version":"1.1.0","ProjectId":"Projects-1"}],"TotalResults":2}`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
			Metadata: Metadata{
				Space: &SpaceMetadata{ID: "Spaces-1", Name: "Default"},
			},
		}

		resources, err := integration.ListResources("release", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Parameters:  map[string]string{"project": "Projects-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "1.0.0", resources[0].Name)
		assert.Equal(t, "Releases-1", resources[0].ID)
		assert.Equal(t, "1.1.0", resources[1].Name)

		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/Spaces-1/projects/Projects-1/releases")
	})

	t.Run("list releases without project parameter -> empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
			Metadata: Metadata{
				Space: &SpaceMetadata{ID: "Spaces-1", Name: "Default"},
			},
		}

		resources, err := integration.ListResources("release", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
			Parameters:  map[string]string{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("unknown resource type -> empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
			Metadata: Metadata{
				Space: &SpaceMetadata{ID: "Spaces-1", Name: "Default"},
			},
		}

		resources, err := integration.ListResources("unknown", core.ListResourcesContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func Test__Octopus__CompareWebhookConfig(t *testing.T) {
	handler := &OctopusWebhookHandler{}

	t.Run("always returns true", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{EventCategories: []string{"DeploymentSucceeded"}},
			WebhookConfiguration{EventCategories: []string{"DeploymentFailed"}},
		)

		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("accepts nil values", func(t *testing.T) {
		equal, err := handler.CompareConfig(nil, nil)

		require.NoError(t, err)
		assert.True(t, equal)
	})
}

func Test__Octopus__MergeWebhookConfig(t *testing.T) {
	handler := &OctopusWebhookHandler{}

	t.Run("merges event categories", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded"},
			},
			WebhookConfiguration{
				EventCategories: []string{"DeploymentFailed"},
			},
		)

		require.NoError(t, err)
		require.True(t, changed)

		config, ok := merged.(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"DeploymentFailed", "DeploymentSucceeded"}, config.EventCategories)
	})

	t.Run("merges projects and environments", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded"},
				Projects:        []string{"Projects-1"},
				Environments:    []string{"Environments-1"},
			},
			WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded"},
				Projects:        []string{"Projects-2"},
				Environments:    []string{"Environments-2"},
			},
		)

		require.NoError(t, err)
		require.True(t, changed)

		config, ok := merged.(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"Projects-1", "Projects-2"}, config.Projects)
		assert.Equal(t, []string{"Environments-1", "Environments-2"}, config.Environments)
	})

	t.Run("no changes when already merged -> changed is false", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded"},
				Projects:        []string{"Projects-1"},
			},
			WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded"},
				Projects:        []string{"Projects-1"},
			},
		)

		require.NoError(t, err)
		assert.False(t, changed)

		config, ok := merged.(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"DeploymentSucceeded"}, config.EventCategories)
		assert.Equal(t, []string{"Projects-1"}, config.Projects)
	})
}

func Test__Octopus__SetupWebhook(t *testing.T) {
	handler := &OctopusWebhookHandler{}

	t.Run("creates subscription and stores metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListSpaces (for spaceIDForIntegration)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
				// ListSubscriptions (for cleanupStaleSubscription)
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
				// CreateSubscription
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"Subscriptions-1","Name":"SuperPlane-wh-123","SpaceId":"Spaces-1"}`,
					)),
				},
			},
		}

		webhookCtx := &integrationWebhookContext{
			id:  "wh-123",
			url: "https://hooks.superplane.dev/octopus",
			configuration: WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded", "DeploymentFailed"},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)

		// Verify secret was stored
		assert.NotEmpty(t, webhookCtx.secret)

		// Verify metadata
		storedMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "Subscriptions-1", storedMetadata.SubscriptionID)
		assert.Equal(t, "Spaces-1", storedMetadata.SpaceID)

		// Verify HTTP requests: ListSpaces + ListSubscriptions + CreateSubscription
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/spaces/all")

		assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.Path, "/api/Spaces-1/subscriptions/all")

		assert.Equal(t, http.MethodPost, httpCtx.Requests[2].Method)
		assert.Contains(t, httpCtx.Requests[2].URL.Path, "/api/Spaces-1/subscriptions")

		// Verify subscription request body
		reqBody, readErr := io.ReadAll(httpCtx.Requests[2].Body)
		require.NoError(t, readErr)

		reqPayload := map[string]any{}
		require.NoError(t, json.Unmarshal(reqBody, &reqPayload))
		assert.Equal(t, "SuperPlane-wh-123", reqPayload["Name"])
		assert.Equal(t, "Spaces-1", reqPayload["SpaceId"])

		sub, ok := reqPayload["EventNotificationSubscription"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://hooks.superplane.dev/octopus", sub["WebhookURI"])
		assert.Equal(t, webhookHeaderKey, sub["WebhookHeaderKey"])
		assert.Equal(t, "00:00:30", sub["WebhookTimeout"])
	})

	t.Run("cleans up stale subscription before creating new one", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// ListSpaces
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
				// ListSubscriptions (returns stale one)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Subscriptions-old","Name":"SuperPlane-wh-456","SpaceId":"Spaces-1"}]`,
					)),
				},
				// DeleteSubscription (cleanup)
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
				// CreateSubscription
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"Id":"Subscriptions-new","Name":"SuperPlane-wh-456","SpaceId":"Spaces-1"}`,
					)),
				},
			},
		}

		webhookCtx := &integrationWebhookContext{
			id:  "wh-456",
			url: "https://hooks.superplane.dev/octopus",
			configuration: WebhookConfiguration{
				EventCategories: []string{"DeploymentSucceeded"},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"serverUrl": "https://octopus.example.com",
				"apiKey":    "API-TEST",
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)

		storedMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "Subscriptions-new", storedMetadata.SubscriptionID)

		// Verify: ListSpaces + ListSubscriptions + DeleteSubscription + CreateSubscription
		require.Len(t, httpCtx.Requests, 4)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[2].Method)
		assert.Contains(t, httpCtx.Requests[2].URL.Path, "/api/Spaces-1/subscriptions/Subscriptions-old")
	})
}

func Test__Octopus__CleanupWebhook(t *testing.T) {
	handler := &OctopusWebhookHandler{}

	t.Run("deletes subscription", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP: httpCtx,
			Webhook: &integrationWebhookContext{
				metadata: WebhookMetadata{
					SubscriptionID: "Subscriptions-1",
					SpaceID:        "Spaces-1",
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/api/Spaces-1/subscriptions/Subscriptions-1")
	})

	t.Run("404 response -> no error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"ErrorMessage":"Not found"}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP: httpCtx,
			Webhook: &integrationWebhookContext{
				metadata: WebhookMetadata{
					SubscriptionID: "Subscriptions-deleted",
					SpaceID:        "Spaces-1",
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("empty metadata -> no-op", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP: httpCtx,
			Webhook: &integrationWebhookContext{
				metadata: WebhookMetadata{},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"serverUrl": "https://octopus.example.com",
					"apiKey":    "API-TEST",
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})
}

func Test__Octopus__resolveSpace(t *testing.T) {
	t.Run("returns default space when no space requested", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true},{"Id":"Spaces-2","Name":"Other","IsDefault":false}]`,
					)),
				},
			},
		}

		client := &Client{
			ServerURL: "https://octopus.example.com",
			APIKey:    "API-TEST",
			http:      httpCtx,
		}

		space, err := resolveSpace(client, "")
		require.NoError(t, err)
		assert.Equal(t, "Spaces-1", space.ID)
		assert.Equal(t, "Default", space.Name)
	})

	t.Run("resolves by name", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true},{"Id":"Spaces-2","Name":"Production","IsDefault":false}]`,
					)),
				},
			},
		}

		client := &Client{
			ServerURL: "https://octopus.example.com",
			APIKey:    "API-TEST",
			http:      httpCtx,
		}

		space, err := resolveSpace(client, "Production")
		require.NoError(t, err)
		assert.Equal(t, "Spaces-2", space.ID)
	})

	t.Run("resolves by ID", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true},{"Id":"Spaces-2","Name":"Production","IsDefault":false}]`,
					)),
				},
			},
		}

		client := &Client{
			ServerURL: "https://octopus.example.com",
			APIKey:    "API-TEST",
			http:      httpCtx,
		}

		space, err := resolveSpace(client, "Spaces-2")
		require.NoError(t, err)
		assert.Equal(t, "Spaces-2", space.ID)
		assert.Equal(t, "Production", space.Name)
	})

	t.Run("space not found -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"Id":"Spaces-1","Name":"Default","IsDefault":true}]`,
					)),
				},
			},
		}

		client := &Client{
			ServerURL: "https://octopus.example.com",
			APIKey:    "API-TEST",
			http:      httpCtx,
		}

		_, err := resolveSpace(client, "NonExistent")
		require.ErrorContains(t, err, "is not accessible")
	})
}
