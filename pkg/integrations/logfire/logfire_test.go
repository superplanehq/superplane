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

func Test__Logfire__Sync__Success(t *testing.T) {
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

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "acme-org", metadata.ExternalOrganizationID)
	assert.True(t, metadata.SupportsQueryAPI)

	assert.Empty(t, integrationCtx.Secrets)

	require.Len(t, httpCtx.Requests, 1)
}

func Test__Logfire__Sync__Fail_InvalidAPIKey(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
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

	require.ErrorContains(t, err, "invalid Logfire API key")
}

func Test__Logfire__Sync__PreservesWebhookSetupFromMetadata(t *testing.T) {
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

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.True(t, metadata.SupportsWebhookSetup)
	assert.True(t, metadata.SupportsQueryAPI)
	assert.Equal(t, "acme-org", metadata.ExternalOrganizationID)
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
