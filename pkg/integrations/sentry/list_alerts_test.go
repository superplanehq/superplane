package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListAlerts__Setup(t *testing.T) {
	component := &ListAlerts{}
	metadata := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"project": "backend",
		},
		Metadata: metadata,
		Integration: &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, ListAlertsNodeMetadata{
		Project: &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
	}, metadata.Metadata)
}

func Test__ListAlerts__Configuration(t *testing.T) {
	component := &ListAlerts{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	assert.Equal(t, ResourceTypeProject, fields[0].TypeOptions.Resource.Type)
}

func Test__ListAlerts__Execute(t *testing.T) {
	component := &ListAlerts{}
	firstPage := sentryMockResponse(
		http.StatusOK,
		`[{"id":"7","name":"High error rate","projects":["backend"],"timeWindow":60.0,"owner":null,"triggers":[{"id":"1","label":"critical","actions":[{"id":"a1","type":"email","targetType":"user","targetIdentifier":"7","inputChannelId":null,"integrationId":null,"sentryAppID":null}]}]},{"id":"8","name":"Latency alert","projects":["frontend"],"timeWindow":30.0,"owner":null}]`,
	)
	firstPage.Header.Set(
		"Link",
		`<https://sentry.io/api/0/organizations/example/alert-rules/?cursor=page2>; rel="next"; results="true"; cursor="page2"`,
	)
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			firstPage,
			sentryMockResponse(http.StatusOK, `[{"id":"9","name":"Backend saturation","projects":["backend"],"timeWindow":15.0,"owner":null}]`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project": "backend",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.alertRules", executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	payload, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	output, ok := payload["data"].(ListAlertsOutput)
	require.True(t, ok)
	require.Len(t, output.Alerts, 2)
	assert.Equal(t, "7", output.Alerts[0].ID)
	assert.Equal(t, "High error rate", output.Alerts[0].Name)
	assert.Equal(t, "9", output.Alerts[1].ID)
	assert.Equal(t, "Backend saturation", output.Alerts[1].Name)
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/?cursor=page2", httpCtx.Requests[1].URL.String())
}
