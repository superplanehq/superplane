package logfire

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Logfire__Sync__MissingAPIKey(t *testing.T) {
	integration := &Logfire{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          &contexts.HTTPContext{},
		Integration:   integrationCtx,
	})

	require.ErrorContains(t, err, "apiKey is required")
}

func Test__Logfire__Sync__Success_FirstUsableProject(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[
						{"id":"proj_123","organization_name":"acme-org","project_name":"backend"},
						{"id":"proj_456","organization_name":"acme-org","project_name":"frontend"}
					]`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"token":"lf_read_token_123"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[],"rows":[]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Metadata: map[string]any{
			"supportsWebhookSetup": true,
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)

	secret, ok := integrationCtx.Secrets[readTokenSecretName]
	require.True(t, ok)
	assert.Equal(t, "lf_read_token_123", string(secret.Value))

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "acme-org", metadata.ExternalOrganizationID)
	assert.Equal(t, "proj_123", metadata.ExternalProjectID)
	assert.True(t, metadata.SupportsWebhookSetup)
	assert.True(t, metadata.SupportsQueryAPI)

	require.Len(t, httpCtx.Requests, 3)
	// CreateReadToken request should target the first usable project.
	// Expected path: /api/v1/projects/<projectID>/read-tokens/
	assert.Contains(t, httpCtx.Requests[1].URL.Path, "/api/v1/projects/proj_123/read-tokens/")
}

func Test__Logfire__Sync__ReusesValidReadTokenSkipsBootstrap(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[],"rows":[]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Metadata: map[string]any{
			"externalOrganizationId": "acme-org",
			"externalProjectId":      "proj_123",
			"supportsWebhookSetup":   true,
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretName: {Name: readTokenSecretName, Value: []byte("existing_read_token")},
		},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/v1/query", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, "Bearer existing_read_token", httpCtx.Requests[0].Header.Get("Authorization"))

	secret := integrationCtx.Secrets[readTokenSecretName]
	assert.Equal(t, "existing_read_token", string(secret.Value))

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "acme-org", metadata.ExternalOrganizationID)
	assert.Equal(t, "proj_123", metadata.ExternalProjectID)
	assert.True(t, metadata.SupportsWebhookSetup)
}

func Test__Logfire__Sync__InvalidStoredToken_FallsBackToBootstrap(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"id":"proj_123","organization_name":"acme-org","project_name":"backend"}]`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"token":"lf_read_token_new"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[],"rows":[]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretName: {Name: readTokenSecretName, Value: []byte("expired_read_token")},
		},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 4)
	assert.Equal(t, "/v1/query", httpCtx.Requests[0].URL.Path)

	secret, ok := integrationCtx.Secrets[readTokenSecretName]
	require.True(t, ok)
	assert.Equal(t, "lf_read_token_new", string(secret.Value))
}

func Test__Logfire__Sync__Success_SecondProjectUsable(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"id":"proj_123","organization_name":"acme-org","project_name":"backend"}]`,
				)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	// Updated mock responses for two projects:
	httpCtx.Responses = []*http.Response{
		{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`[
					{"id":"proj_123","organization_name":"acme-org","project_name":"backend"},
					{"id":"proj_456","organization_name":"acme-org","project_name":"frontend"}
				]`,
			)),
		},
		{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"detail":"forbidden"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"token":"lf_read_token_456"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"columns":[],"rows":[]}`)),
		},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)

	secret, ok := integrationCtx.Secrets[readTokenSecretName]
	require.True(t, ok)
	assert.Equal(t, "lf_read_token_456", string(secret.Value))

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "acme-org", metadata.ExternalOrganizationID)
	assert.Equal(t, "proj_456", metadata.ExternalProjectID)

	require.Len(t, httpCtx.Requests, 4)
	assert.Contains(t, httpCtx.Requests[1].URL.Path, "/api/v1/projects/proj_123/read-tokens/")
	assert.Contains(t, httpCtx.Requests[2].URL.Path, "/api/v1/projects/proj_456/read-tokens/")
}

func Test__Logfire__Sync__Fail_NoPermissionAnyProject(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[
						{"id":"proj_123","organization_name":"acme-org","project_name":"backend"},
						{"id":"proj_456","organization_name":"acme-org","project_name":"frontend"}
					]`,
				)),
			},
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"detail":"forbidden"}`)),
			},
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"detail":"forbidden"}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})
	require.ErrorContains(t, err, "api key has no permission to create read tokens in any accessible project")
}

func Test__Logfire__ListResources__Projects(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[
						{"id":"proj_123","organization_name":"acme-org","project_name":"backend"},
						{"id":"proj_456","organization_name":"acme-org","project_name":"frontend"}
					]`,
				)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	resources, err := integration.ListResources("project", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})
	require.NoError(t, err)

	require.Len(t, resources, 2)
	assert.Equal(t, "project", resources[0].Type)
	assert.Equal(t, "backend", resources[0].Name)
	assert.Equal(t, "proj_123", resources[0].ID)
	assert.Equal(t, "project", resources[1].Type)
	assert.Equal(t, "frontend", resources[1].Name)
	assert.Equal(t, "proj_456", resources[1].ID)
}

func Test__Logfire__ListResources__Alerts__MissingProjectId_ReturnsEmptyNoError(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	resources, err := integration.ListResources("alert", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})
	require.NoError(t, err)
	require.Len(t, resources, 0)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__Logfire__ListResources__Alerts__WithProjectId_Undefined_ReturnsEmptyNoError(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	resources, err := integration.ListResources("alert", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Parameters: map[string]string{
			"projectId": "undefined",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 0)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__Logfire__ListResources__Alerts__WithProjectId_Null_ReturnsEmptyNoError(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	resources, err := integration.ListResources("alert", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Parameters: map[string]string{
			"projectId": "null",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 0)
	require.Len(t, httpCtx.Requests, 0)
}

func Test__Logfire__ListResources__Alerts__WithProjectId(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[
						{"id":"alt_1","name":"Latency spike"},
						{"alert_id":"alt_2","alert_name":"Errors spike"}
					]`,
				)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	resources, err := integration.ListResources("alert", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Parameters: map[string]string{
			"projectId": "proj_123",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "alt_1", resources[0].ID)
	assert.Equal(t, "Latency spike", resources[0].Name)
	assert.Equal(t, "alt_2", resources[1].ID)
	assert.Equal(t, "Errors spike", resources[1].Name)
	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1/projects/proj_123/alerts/")
}
