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

func Test__GetAlert__Setup(t *testing.T) {
	component := &GetAlert{}
	metadata := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"project": "backend",
			"alertId": "7",
		},
		Metadata: metadata,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{Slug: "example"},
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","projects":["backend"]}`),
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, GetAlertNodeMetadata{
		Project:   &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
		AlertName: "High error rate · backend",
	}, metadata.Metadata)
}

func Test__GetAlert__Setup__SkipsAPIWhenAlertIDIsExpression(t *testing.T) {
	component := &GetAlert{}
	metadata := &contexts.MetadataContext{}
	httpCtx := &contexts.HTTPContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"project": "backend",
			"alertId": "{{ steps.trigger.output.alertId }}",
		},
		Metadata: metadata,
		Integration: &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		},
		HTTP: httpCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, GetAlertNodeMetadata{
		Project:   &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
		AlertName: "",
	}, metadata.Metadata)
	assert.Empty(t, httpCtx.Requests)
}

func Test__GetAlert__Configuration(t *testing.T) {
	component := &GetAlert{}
	fields := component.Configuration()

	require.Len(t, fields, 2)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
	require.NotNil(t, fields[1].TypeOptions)
	require.NotNil(t, fields[1].TypeOptions.Resource)
	assert.Equal(t, ResourceTypeAlert, fields[1].TypeOptions.Resource.Type)
	require.Len(t, fields[1].TypeOptions.Resource.Parameters, 1)
	assert.Equal(t, "project", fields[1].TypeOptions.Resource.Parameters[0].Name)
}

func Test__GetAlert__Execute(t *testing.T) {
	component := &GetAlert{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","aggregate":"count()","environment":"prod","timeWindow":30.0,"owner":null,"projects":["backend"],"triggers":[{"id":"1","label":"critical","actions":[{"id":"a1","type":"email","targetType":"user","targetIdentifier":"7","inputChannelId":null,"integrationId":null,"sentryAppId":null}]}],"dateCreated":"2026-03-25T10:00:00Z","dateModified":"2026-03-25T10:05:00Z"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project": "backend",
			"alertId": "7",
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
	assert.Equal(t, "sentry.alertRule", executionState.Type)
	require.Len(t, executionState.Payloads, 1)

	payload, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	output, ok := payload["data"].(*MetricAlertRule)
	require.True(t, ok)
	assert.Equal(t, "7", output.ID)
	assert.Equal(t, "High error rate", output.Name)
}
