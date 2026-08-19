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

func Test__GetIssue__Setup(t *testing.T) {
	component := GetIssue{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"issueKey": "TEST-123",
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing issueKey -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project": "TEST",
			},
		})

		require.ErrorContains(t, err, "issueKey is required")
	})

	t.Run("project not found -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "OTHER", Name: "Other"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-123",
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
				"project":  "TEST",
				"issueKey": "TEST-123",
			},
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, "TEST", nodeMetadata.Project.Key)
	})
}

func Test__GetIssue__Execute(t *testing.T) {
	component := GetIssue{}

	t.Run("successful get issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-123","fields":{"summary":"hi"}}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-123",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, GetIssuePayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/issue/TEST-123"))
	})

	t.Run("expand parameter is forwarded", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-123"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-123",
				"expand":   "renderedFields,names",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "expand=renderedFields%2Cnames")
	})

	t.Run("issue not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Issue does not exist"]}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "MISSING-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get issue")
	})
}

func Test__GetIssue__ComponentInfo(t *testing.T) {
	component := GetIssue{}

	assert.Equal(t, "jira.getIssue", component.Name())
	assert.Equal(t, "Get Issue", component.Label())
	assert.Equal(t, "jira", component.Icon())
	assert.NotEmpty(t, component.Documentation())
}

func Test__GetIssue__Configuration(t *testing.T) {
	component := GetIssue{}

	config := component.Configuration()
	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "project")
	assert.Contains(t, fieldNames, "issueKey")
	assert.Contains(t, fieldNames, "expand")
}

func Test__GetIssue__OutputChannels(t *testing.T) {
	component := GetIssue{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}
