package sentry

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateAlert__Setup(t *testing.T) {
	component := &CreateAlert{}
	metadata := &contexts.MetadataContext{}
	criticalThreshold := 5.0

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"project":    "backend",
			"name":       "High error rate",
			"aggregate":  "count()",
			"timeWindow": 60.0,
			"critical": map[string]any{
				"threshold": criticalThreshold,
				"notification": map[string]any{
					"targetType":       "user",
					"targetIdentifier": "4346509",
				},
			},
		},
		Metadata: metadata,
		Integration: &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{{ID: "1", Slug: "backend", Name: "Backend"}},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AlertRuleNodeMetadata{
		Project: &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
	}, metadata.Metadata)
}

func Test__CreateAlert__Configuration(t *testing.T) {
	component := &CreateAlert{}
	fields := component.Configuration()

	require.Len(t, fields, 10)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.Equal(t, "critical", fields[8].Name)
	assert.Equal(t, configuration.FieldTypeObject, fields[8].Type)
	assert.Equal(t, "warning", fields[9].Name)
	criticalSchema := fields[8].TypeOptions.Object.Schema
	notificationSchema := criticalSchema[2].TypeOptions.Object.Schema
	assert.Equal(t, "targetIdentifier", notificationSchema[1].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, notificationSchema[1].Type)
}

func Test__CreateAlert__Execute(t *testing.T) {
	component := &CreateAlert{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","aggregate":"count()","timeWindow":60.0,"query":"","environment":"production","projects":["backend"],"eventTypes":["default","error"],"triggers":[{"id":"1","label":"critical","alertThreshold":5.0,"resolveThreshold":2.0,"actions":[{"id":"a1","type":"email","targetType":"user","targetIdentifier":"4346509","inputChannelId":null,"integrationId":null,"sentryAppId":null,"priority":null}]}]}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}
	resolveThreshold := 2.0

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project":       "backend",
			"name":          "High error rate",
			"aggregate":     "count()",
			"timeWindow":    60.0,
			"thresholdType": "above",
			"environment":   "production",
			"eventTypes":    []string{"default", "error"},
			"critical": map[string]any{
				"threshold":        5.0,
				"resolveThreshold": resolveThreshold,
				"notification": map[string]any{
					"targetType":       "user",
					"targetIdentifier": "4346509",
				},
			},
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://sentry.io", "userToken": "user-token"},
			Metadata:      Metadata{Organization: &OrganizationSummary{Slug: "example"}},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/", httpCtx.Requests[0].URL.String())

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)

	var request CreateOrUpdateMetricAlertRuleRequest
	require.NoError(t, json.Unmarshal(body, &request))
	assert.Equal(t, "High error rate", request.Name)
	assert.Equal(t, []string{"backend"}, request.Projects)
	assert.Equal(t, []string{"default", "error"}, request.EventTypes)
	require.Len(t, request.Triggers, 1)
	assert.Equal(t, alertTriggerLabelCritical, request.Triggers[0].Label)
	assert.Equal(t, 5.0, request.Triggers[0].AlertThreshold)
	require.NotNil(t, request.Triggers[0].ResolveThreshold)
	assert.Equal(t, 2.0, *request.Triggers[0].ResolveThreshold)

	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.alertRule", executionState.Type)
}
