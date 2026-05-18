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

func strPtr(v string) *string { return &v }

func Test__UpdateIssue__Setup(t *testing.T) {
	component := UpdateIssue{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"issueKey": "TEST-1",
				"summary":  "X",
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
				"summary": "X",
			},
		})

		require.ErrorContains(t, err, "issueKey is required")
	})

	t.Run("no update fields -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
		})

		require.ErrorContains(t, err, "at least one field to update")
	})

	t.Run("project not found -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "OTHER", Name: "Other"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
				"summary":  "New",
			},
		})

		require.ErrorContains(t, err, "project TEST not found")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
				"summary":  "New summary",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	component := UpdateIssue{}

	t.Run("successful update + refetch", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1","fields":{"summary":"New"}}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
				"summary":  "New",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, UpdateIssuePayloadType, execCtx.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)

		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		fields, ok := payload["fields"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "New", fields["summary"])
	})

	t.Run("update with assignee unassign", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1"}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
				"assignee": "-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)

		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &payload))
		fields := payload["fields"].(map[string]any)
		assert.Contains(t, fields, "assignee")
		assert.Nil(t, fields["assignee"])
	})

	t.Run("notifyUsers param forwarded", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(``))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1"}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "TEST",
				"issueKey":    "TEST-1",
				"summary":     "X",
				"notifyUsers": false,
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "notifyUsers=false")
	})

	t.Run("update failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["bad"]}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
				"summary":  "X",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update issue")
	})
}

func Test__UpdateIssue__BuildFields(t *testing.T) {
	t.Run("description empty string clears field", func(t *testing.T) {
		spec := UpdateIssueSpec{Description: strPtr("")}
		fields := buildUpdateFields(spec)
		_, present := fields["description"]
		assert.True(t, present)
		assert.Nil(t, fields["description"])
	})

	t.Run("description sets adf doc", func(t *testing.T) {
		spec := UpdateIssueSpec{Description: strPtr("Hi")}
		fields := buildUpdateFields(spec)
		assert.NotNil(t, fields["description"])
	})

	t.Run("labels nil pointer -> not in fields", func(t *testing.T) {
		spec := UpdateIssueSpec{}
		fields := buildUpdateFields(spec)
		_, present := fields["labels"]
		assert.False(t, present)
	})

	t.Run("labels empty list -> empty slice in fields", func(t *testing.T) {
		empty := []string{}
		spec := UpdateIssueSpec{Labels: &empty}
		fields := buildUpdateFields(spec)
		labels := fields["labels"].([]string)
		assert.Empty(t, labels)
	})
}

func Test__UpdateIssue__ComponentInfo(t *testing.T) {
	component := UpdateIssue{}
	assert.Equal(t, "jira.updateIssue", component.Name())
	assert.Equal(t, "Update Issue", component.Label())
	assert.NotEmpty(t, component.Documentation())
}
