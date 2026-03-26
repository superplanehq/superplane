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

func Test__CreateRelease__Setup(t *testing.T) {
	component := &CreateRelease{}

	t.Run("stores selected project metadata after validating release access", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "backend",
				"version": "2026.03.25",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					sentryMockResponse(http.StatusOK, `[]`),
				},
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
					Projects: []ProjectSummary{
						{ID: "1", Slug: "backend", Name: "Backend"},
					},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, CreateReleaseNodeMetadata{
			Project: &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
		}, metadata.Metadata)
	})

	t.Run("fails with release scope guidance when token cannot access releases", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "backend",
				"version": "2026.03.25",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					sentryMockResponse(http.StatusForbidden, `{"detail":"You do not have permission to perform this action."}`),
				},
			},
			Metadata: &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseUrl":   "https://sentry.io",
					"userToken": "user-token",
				},
				Metadata: Metadata{
					Organization: &OrganizationSummary{
						Slug: "example",
					},
					Projects: []ProjectSummary{
						{ID: "1", Slug: "backend", Name: "Backend"},
					},
				},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), releaseScope)
	})
}

func Test__CreateRelease__Configuration(t *testing.T) {
	component := &CreateRelease{}
	fields := component.Configuration()

	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.Equal(t, configuration.FieldTypeString, fields[1].Type)
	assert.Equal(t, configuration.FieldTypeString, fields[2].Type)
	assert.Equal(t, configuration.FieldTypeList, fields[4].Type)
	assert.Equal(t, configuration.FieldTypeList, fields[5].Type)
}

func Test__CreateRelease__Execute(t *testing.T) {
	component := &CreateRelease{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, `{"id":2,"version":"2026.03.25","shortVersion":"2026.03.25","ref":"abc123","dateCreated":"2026-03-25T10:00:00Z","commitCount":1,"deployCount":0,"newGroups":0,"projects":[{"name":"Backend","slug":"backend"}]}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project": "backend",
			"version": "2026.03.25",
			"ref":     "abc123",
			"url":     "https://example.com/releases/2026.03.25",
			"commits": []map[string]any{
				{
					"id":          "abc123",
					"repository":  "superplanehq/superplane",
					"message":     "Ship sentry actions",
					"authorName":  "Washington",
					"authorEmail": "washington@example.com",
					"timestamp":   "2026-03-25T09:55:00Z",
				},
			},
			"refs": []map[string]any{
				{
					"repository":     "superplanehq/superplane",
					"commit":         "abc123",
					"previousCommit": "def456",
				},
			},
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
	assert.Equal(t, "sentry.release", executionState.Type)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/releases/", httpCtx.Requests[0].URL.String())

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "2026.03.25", requestBody["version"])
	assert.Equal(t, "abc123", requestBody["ref"])
	assert.Equal(t, "https://example.com/releases/2026.03.25", requestBody["url"])
	_, hasDateReleased := requestBody["dateReleased"]
	assert.False(t, hasDateReleased)
	assert.Equal(t, []any{"backend"}, requestBody["projects"])
	require.Len(t, requestBody["commits"], 1)
	require.Len(t, requestBody["refs"], 1)
}
