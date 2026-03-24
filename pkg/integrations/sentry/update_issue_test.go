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

func Test__UpdateIssue__Setup(t *testing.T) {
	component := &UpdateIssue{}

	t.Run("requires status", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "123",
			},
		})

		require.ErrorContains(t, err, "status is required")
	})

	t.Run("accepts status update", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "123",
				"status":  "resolved",
			},
			Metadata: metadata,
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
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout"}`),
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, UpdateIssueNodeMetadata{IssueTitle: "EXAMPLE-1 · RuntimeError"}, metadata.Metadata)
	})

	t.Run("uses issue resources", func(t *testing.T) {
		issueField := component.Configuration()[0]

		assert.Equal(t, configuration.FieldTypeIntegrationResource, issueField.Type)
		require.NotNil(t, issueField.TypeOptions)
		require.NotNil(t, issueField.TypeOptions.Resource)
		assert.Equal(t, ResourceTypeIssue, issueField.TypeOptions.Resource.Type)
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	component := &UpdateIssue{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":   "https://sentry.io",
			"userToken": "user-token",
		},
		Metadata: Metadata{
			Organization: &OrganizationSummary{
				Slug: "example",
			},
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"123","status":"resolved","assignedTo":{"name":"Platform"}}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId": "123",
			"status":  "resolved",
		},
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.issue", executionState.Type)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/", httpCtx.Requests[0].URL.String())

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "resolved", requestBody["status"])
	_, hasAssignedTo := requestBody["assignedTo"]
	assert.False(t, hasAssignedTo)
}
