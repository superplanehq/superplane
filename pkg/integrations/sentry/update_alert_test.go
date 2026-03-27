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

func Test__UpdateAlert__Setup(t *testing.T) {
	component := &UpdateAlert{}
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

func Test__UpdateAlert__Configuration(t *testing.T) {
	component := &UpdateAlert{}
	fields := component.Configuration()

	require.Len(t, fields, 11)
	assert.Equal(t, "project", fields[0].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.Equal(t, "alertId", fields[1].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
	assert.Equal(t, "critical", fields[9].Name)
	assert.Equal(t, "warning", fields[10].Name)
	criticalSchema := fields[9].TypeOptions.Object.Schema
	notificationSchema := criticalSchema[2].TypeOptions.Object.Schema
	assert.Equal(t, "targetIdentifier", notificationSchema[1].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, notificationSchema[1].Type)
}

func Test__UpdateAlert__Execute(t *testing.T) {
	component := &UpdateAlert{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","aggregate":"count()","query":"","thresholdType":0,"timeWindow":60.0,"environment":"production","projects":["backend"],"eventTypes":["default","error"],"triggers":[{"id":"1","label":"critical","alertThreshold":5.0,"resolveThreshold":2.0,"actions":[{"id":"a1","type":"email","targetType":"user","targetIdentifier":"4346509","inputChannelId":null,"integrationId":null,"sentryAppId":null,"priority":null}]}]}`),
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"High error rate","aggregate":"count()","query":"level:error","thresholdType":0,"timeWindow":30.0,"environment":"production","projects":["backend"],"eventTypes":["error"],"triggers":[{"id":"1","label":"critical","alertThreshold":10.0,"resolveThreshold":3.0,"actions":[{"id":"a1","type":"email","targetType":"team","targetIdentifier":"42","inputChannelId":null,"integrationId":null,"sentryAppId":null,"priority":null}]}]}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}
	timeWindow := 30.0
	resolveThreshold := 3.0

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertId":    "7",
			"query":      "level:error",
			"timeWindow": timeWindow,
			"eventTypes": []string{"error"},
			"critical": map[string]any{
				"threshold":        10.0,
				"resolveThreshold": resolveThreshold,
				"notification": map[string]any{
					"targetType":       "team",
					"targetIdentifier": "42",
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
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/7/", httpCtx.Requests[1].URL.String())

	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)

	var request CreateOrUpdateMetricAlertRuleRequest
	require.NoError(t, json.Unmarshal(body, &request))
	assert.Equal(t, "High error rate", request.Name)
	assert.Equal(t, "level:error", request.Query)
	assert.Equal(t, 30, request.TimeWindow)
	assert.Equal(t, []string{"error"}, request.EventTypes)
	require.Len(t, request.Triggers, 1)
	assert.Equal(t, 10.0, request.Triggers[0].AlertThreshold)
	assert.Equal(t, "team", request.Triggers[0].Actions[0].TargetType)
	assert.Equal(t, "42", request.Triggers[0].Actions[0].TargetIdentifier)

	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.alertRule", executionState.Type)
}

func Test__UpdateAlert__Execute__PreservesEventTypesWhenOmitted(t *testing.T) {
	component := &UpdateAlert{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"Only error types","aggregate":"count()","query":"","thresholdType":0,"timeWindow":60.0,"environment":"","projects":["backend"],"eventTypes":["error"],"triggers":[{"id":"1","label":"critical","alertThreshold":5.0,"actions":[{"id":"a1","type":"email","targetType":"user","targetIdentifier":"1","inputChannelId":null,"integrationId":null,"sentryAppId":null,"priority":null}]}]}`),
			sentryMockResponse(http.StatusOK, `{"id":"7","name":"Only error types","aggregate":"count()","query":"new","thresholdType":0,"timeWindow":60.0,"environment":"","projects":["backend"],"eventTypes":["error"],"triggers":[{"id":"1","label":"critical","alertThreshold":5.0,"actions":[{"id":"a1","type":"email","targetType":"user","targetIdentifier":"1","inputChannelId":null,"integrationId":null,"sentryAppId":null,"priority":null}]}]}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertId": "7",
			"query":   "new",
			"critical": map[string]any{
				"threshold": 5.0,
				"notification": map[string]any{
					"targetType":       "user",
					"targetIdentifier": "1",
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
	require.Len(t, httpCtx.Requests, 2)

	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)

	var request CreateOrUpdateMetricAlertRuleRequest
	require.NoError(t, json.Unmarshal(body, &request))
	assert.Equal(t, []string{"error"}, request.EventTypes)
}
