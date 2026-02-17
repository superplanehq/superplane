package jira

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

func Test__CreateIssue__Setup(t *testing.T) {
	component := CreateIssue{}

	t.Run("missing project -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"issueType": "Task",
				"summary":   "Test summary",
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing issueType -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project": "TEST",
				"summary": "Test summary",
			},
		})

		require.ErrorContains(t, err, "issueType is required")
	})

	t.Run("missing summary -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
			},
		})

		require.ErrorContains(t, err, "summary is required")
	})

	t.Run("project not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"OTHER","name":"Other Project"}]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Test summary",
			},
		})

		require.ErrorContains(t, err, "project TEST not found")
	})

	t.Run("valid setup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Test summary",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateIssue__Execute(t *testing.T) {
	component := CreateIssue{}

	t.Run("successful issue creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-123","self":"https://test.atlassian.net/rest/api/3/issue/10001"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "New task",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, CreateIssuePayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("successful issue creation with description", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10002","key":"TEST-124","self":"https://test.atlassian.net/rest/api/3/issue/10002"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "TEST",
				"issueType":   "Bug",
				"summary":     "Bug report",
				"description": "This is a detailed bug description",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
	})

	t.Run("issue creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Invalid issue type"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "InvalidType",
				"summary":   "Test",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create issue")
	})
}

func Test__CreateIssue__ComponentInfo(t *testing.T) {
	component := CreateIssue{}

	assert.Equal(t, "jira.createIssue", component.Name())
	assert.Equal(t, "Create Issue", component.Label())
	assert.Equal(t, "Create a new issue in Jira", component.Description())
	assert.Equal(t, "jira", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Documentation())
}

func Test__CreateIssue__Configuration(t *testing.T) {
	component := CreateIssue{}

	config := component.Configuration()
	assert.Len(t, config, 4)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "project")
	assert.Contains(t, fieldNames, "issueType")
	assert.Contains(t, fieldNames, "summary")
	assert.Contains(t, fieldNames, "description")

	for _, f := range config {
		if f.Name == "description" {
			assert.False(t, f.Required, "description should be optional")
		} else {
			assert.True(t, f.Required, "%s should be required", f.Name)
		}
	}
}

func Test__CreateIssue__OutputChannels(t *testing.T) {
	component := CreateIssue{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}
