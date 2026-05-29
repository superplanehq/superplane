package jira

import (
	"encoding/json"
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
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test Project"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"issueType": "Task",
				"summary":   "Test summary",
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing issueType -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test Project"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project": "TEST",
				"summary": "Test summary",
			},
		})

		require.ErrorContains(t, err, "issueType is required")
	})

	t.Run("missing summary -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test Project"}},
			}),
			Metadata: &contexts.MetadataContext{},
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

		err := component.Setup(core.SetupContext{
			HTTP: httpContext,
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "OTHER", Name: "Other Project"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Test summary",
			},
		})

		require.ErrorContains(t, err, "project TEST not found")
	})

	t.Run("valid setup", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test Project"}},
			}),
			Metadata: metadataCtx,
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Test summary",
			},
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, "TEST", nodeMetadata.Project.Key)
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
				// post-create GetIssue fetch
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-123","fields":{"summary":"New task","status":{"name":"To Do"}}}`)),
				},
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
			Integration:    newAuthorizedIntegration(),
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
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10002","key":"TEST-124","fields":{"summary":"Bug report","status":{"name":"To Do"}}}`)),
				},
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
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
	})

	t.Run("successful issue creation with assignee", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10003","key":"TEST-125","self":"https://test.atlassian.net/rest/api/3/issue/10003"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10003","key":"TEST-125","fields":{"summary":"Assigned task","assignee":{"accountId":"acct-123","displayName":"Alice"}}}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Assigned task",
				"assignee":  "acct-123",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)

		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		var request map[string]any
		require.NoError(t, json.Unmarshal(body, &request))
		fields := request["fields"].(map[string]any)
		assignee := fields["assignee"].(map[string]any)
		assert.Equal(t, "acct-123", assignee["accountId"])
	})

	t.Run("successful issue creation with status transition", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// create
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10003","key":"TEST-125","self":"https://test.atlassian.net/rest/api/3/issue/10003"}`)),
				},
				// list transitions
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"11","name":"Start","to":{"id":"3","name":"In Progress"}}]}`)),
				},
				// execute transition (204 No Content)
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				// final get
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10003","key":"TEST-125","fields":{"status":{"name":"In Progress"}}}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Move me",
				"status":    "In Progress",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, http.MethodPost, httpContext.Requests[2].Method)
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/rest/api/3/issue/TEST-125/transitions")
	})

	t.Run("status with no matching transition -> error mentioning available targets", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10004","key":"TEST-126"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"11","name":"Start","to":{"id":"3","name":"In Progress"}}]}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "Task",
				"summary":   "Move me",
				"status":    "Done",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "TEST-126")
		assert.Contains(t, err.Error(), "Done")
		assert.Contains(t, err.Error(), "In Progress")
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

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "TEST",
				"issueType": "InvalidType",
				"summary":   "Test",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
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
	assert.Len(t, config, 6)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "project")
	assert.Contains(t, fieldNames, "issueType")
	assert.Contains(t, fieldNames, "summary")
	assert.Contains(t, fieldNames, "description")
	assert.Contains(t, fieldNames, "assignee")
	assert.Contains(t, fieldNames, "status")

	optionalFields := map[string]bool{"description": true, "assignee": true, "status": true}
	for _, f := range config {
		if optionalFields[f.Name] {
			assert.False(t, f.Required, "%s should be optional", f.Name)
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
