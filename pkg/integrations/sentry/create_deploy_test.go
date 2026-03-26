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

func Test__CreateDeploy__Setup(t *testing.T) {
	component := &CreateDeploy{}

	t.Run("stores selected project metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":        "backend",
				"releaseVersion": "2026.03.25",
				"environment":    "production",
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
		assert.Equal(t, CreateDeployNodeMetadata{
			Project: &ProjectSummary{ID: "1", Slug: "backend", Name: "Backend"},
		}, metadata.Metadata)
	})

	t.Run("allows deploy without project", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"releaseVersion": "2026.03.25",
				"environment":    "production",
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
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, CreateDeployNodeMetadata{}, metadata.Metadata)
	})

	t.Run("fails with release scope guidance when token cannot access releases", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"releaseVersion": "2026.03.25",
				"environment":    "production",
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
				},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), releaseScope)
	})
}

func Test__CreateDeploy__Configuration(t *testing.T) {
	component := &CreateDeploy{}
	fields := component.Configuration()

	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
	require.NotNil(t, fields[1].TypeOptions)
	require.NotNil(t, fields[1].TypeOptions.Resource)
	assert.Equal(t, ResourceTypeRelease, fields[1].TypeOptions.Resource.Type)
	require.Len(t, fields[1].TypeOptions.Resource.Parameters, 1)
	assert.Equal(t, "project", fields[1].TypeOptions.Resource.Parameters[0].Name)
	assert.Equal(t, configuration.FieldTypeDateTime, fields[5].Type)
	assert.Equal(t, configuration.FieldTypeDateTime, fields[6].Type)
}

func Test__CreateDeploy__Execute(t *testing.T) {
	component := &CreateDeploy{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, `{"environment":"production","name":"Deploy #42","url":"https://example.com/deploys/42","dateStarted":"2026-03-25T10:00:00Z","dateFinished":"2026-03-25T10:05:00Z","id":"1234567"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"project":        "backend",
			"releaseVersion": "2026.03.25",
			"environment":    "production",
			"name":           "Deploy #42",
			"url":            "https://example.com/deploys/42",
			"dateStarted":    "2026-03-25T10:00:00Z",
			"dateFinished":   "2026-03-25T10:05:00Z",
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
	assert.Equal(t, "sentry.deploy", executionState.Type)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/releases/2026.03.25/deploys/", httpCtx.Requests[0].URL.String())

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "production", requestBody["environment"])
	assert.Equal(t, "Deploy #42", requestBody["name"])
	assert.Equal(t, "https://example.com/deploys/42", requestBody["url"])
	assert.Equal(t, "2026-03-25T10:00:00Z", requestBody["dateStarted"])
	assert.Equal(t, "2026-03-25T10:05:00Z", requestBody["dateFinished"])
	assert.Equal(t, []any{"backend"}, requestBody["projects"])
}
