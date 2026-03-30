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

func Test__DeleteAlert__Setup(t *testing.T) {
	component := &DeleteAlert{}
	metadata := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"project": "backend",
			"alertId": "7",
		},
		Metadata: metadata,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "user-token"},
			Metadata: Metadata{
				Organization: &OrganizationSummary{Slug: "example"},
				Projects:     []ProjectSummary{{ID: "1", Slug: "backend", Name: "Backend"}},
			},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","projects":["backend"]}`),
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AlertRuleNodeMetadata{
		Project:   &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
		AlertName: "High error rate · backend",
	}, metadata.Metadata)
}

func Test__DeleteAlert__Configuration(t *testing.T) {
	component := &DeleteAlert{}
	fields := component.Configuration()

	require.Len(t, fields, 2)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
}

func Test__DeleteAlert__Execute(t *testing.T) {
	component := &DeleteAlert{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","projects":["backend"]}`),
			sentryMockResponse(http.StatusNoContent, ``),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{"alertId": "7"},
		HTTP:          httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "user-token"},
			Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/7/", httpCtx.Requests[1].URL.String())
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[1].Method)

	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.alertDeleted", executionState.Type)
}
