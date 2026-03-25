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

	t.Run("requires at least one update field", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "123",
			},
		})

		require.ErrorContains(t, err, "at least one field to update must be provided")
	})

	t.Run("accepts priority and assignee update", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId":    "123",
				"priority":   "high",
				"assignedTo": "team:42",
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
					sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout","project":{"slug":"backend","name":"Backend"}}`),
					sentryMockResponse(http.StatusOK, `[{"id":"7","name":"Alice Jones","email":"alice@example.com","user":{"id":"7","name":"Alice Jones","email":"alice@example.com","username":"alice"}}]`),
					sentryMockResponse(http.StatusOK, `[{"id":"42","slug":"platform","name":"Platform"}]`),
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, UpdateIssueNodeMetadata{
			IssueTitle:    "EXAMPLE-1 · Database timeout",
			AssigneeLabel: "Team · Platform",
		}, metadata.Metadata)
	})

	t.Run("uses issue resources", func(t *testing.T) {
		issueField := component.Configuration()[0]

		assert.Equal(t, configuration.FieldTypeIntegrationResource, issueField.Type)
		require.NotNil(t, issueField.TypeOptions)
		require.NotNil(t, issueField.TypeOptions.Resource)
		assert.Equal(t, ResourceTypeIssue, issueField.TypeOptions.Resource.Type)
	})

	t.Run("uses assignee resources scoped by issue", func(t *testing.T) {
		assigneeField := component.Configuration()[3]

		assert.Equal(t, configuration.FieldTypeIntegrationResource, assigneeField.Type)
		require.NotNil(t, assigneeField.TypeOptions)
		require.NotNil(t, assigneeField.TypeOptions.Resource)
		assert.Equal(t, ResourceTypeAssignee, assigneeField.TypeOptions.Resource.Type)
		require.Len(t, assigneeField.TypeOptions.Resource.Parameters, 1)
		require.NotNil(t, assigneeField.TypeOptions.Resource.Parameters[0].ValueFrom)
		assert.Equal(t, "issueId", assigneeField.TypeOptions.Resource.Parameters[0].ValueFrom.Field)
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
			sentryMockResponse(http.StatusOK, `{"status":"resolved","assignedTo":{"name":"Platform"},"priority":"high","hasSeen":true,"isPublic":true,"isSubscribed":false}`),
			sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout","status":"resolved","priority":"high","hasSeen":true,"isPublic":true,"isSubscribed":false,"assignedTo":{"name":"Platform"},"project":{"name":"Backend","slug":"backend"}}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId":      "123",
			"status":       "resolved",
			"priority":     "high",
			"assignedTo":   "team:42",
			"hasSeen":      true,
			"isPublic":     true,
			"isSubscribed": false,
		},
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.issue", executionState.Type)
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/?id=123", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/", httpCtx.Requests[1].URL.String())

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "resolved", requestBody["status"])
	assert.Equal(t, "team:42", requestBody["assignedTo"])
	assert.Equal(t, "high", requestBody["priority"])
	assert.Equal(t, true, requestBody["hasSeen"])
	assert.Equal(t, true, requestBody["isPublic"])
	assert.Equal(t, false, requestBody["isSubscribed"])
}

func Test__UpdateIssue__Execute_AcceptsLegacyStringBooleans(t *testing.T) {
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
			sentryMockResponse(http.StatusOK, `{"status":"resolved","hasSeen":true,"isPublic":true,"isSubscribed":false}`),
			sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout","status":"resolved","hasSeen":true,"isPublic":true,"isSubscribed":false}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId":      "123",
			"status":       "resolved",
			"hasSeen":      "true",
			"isPublic":     "true",
			"isSubscribed": "false",
		},
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	require.Len(t, httpCtx.Requests, 2)

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, true, requestBody["hasSeen"])
	assert.Equal(t, true, requestBody["isPublic"])
	assert.Equal(t, false, requestBody["isSubscribed"])
}
