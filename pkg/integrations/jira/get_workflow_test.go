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

func Test__GetWorkflow__Setup(t *testing.T) {
	component := GetWorkflow{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueKey": "TEST-1"},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing issue key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "TEST"},
		})

		require.ErrorContains(t, err, "issueKey is required")
	})

	t.Run("valid setup stores project metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{ID: "10000", Key: "TEST", Name: "Test Project"}},
			}),
			Metadata: metadataCtx,
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, "TEST", nodeMetadata.Project.Key)
	})
}

func Test__GetWorkflow__Execute(t *testing.T) {
	component := GetWorkflow{}

	const issueResponse = `{
		"id":"10001","key":"TEST-1","self":"https://test.atlassian.net/rest/api/3/issue/10001",
		"fields":{
			"status":{"id":"10002","name":"In Progress"},
			"issuetype":{"name":"Task"},
			"project":{"id":"10000","key":"TEST"}
		}
	}`
	const transitionsResponse = `{"transitions":[
		{"id":"31","name":"Resolve","to":{"id":"10003","name":"Done"}},
		{"id":"21","name":"Back to To Do","to":{"id":"10001","name":"To Do"}}
	]}`
	const projectSchemeResponse = `{"values":[{"projectIds":["10000"],"workflowScheme":{"id":"101010","name":"Default scheme"}}]}`
	const schemeDetailResponse = `{"id":101010,"name":"Default scheme","defaultWorkflow":"wf","issueTypeMappings":{"10100":"task-workflow"}}`
	const issueTypesResponse = `{"issueTypes":[{"id":"10100","name":"Task"}]}`
	const workflowStatusesResponse = `{"values":[{"id":{"name":"task-workflow"},"statuses":[
		{"id":"10001","name":"To Do","statusCategory":"TODO"},
		{"id":"10002","name":"In Progress","statusCategory":"IN_PROGRESS"},
		{"id":"10003","name":"Done","statusCategory":"DONE"}
	]}]}`

	t.Run("returns workflow + current status + transitions for a company-managed project", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectSchemeResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(schemeDetailResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueTypesResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(workflowStatusesResponse))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
			Logger:         newLogger(),
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, GetWorkflowPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		output := unwrapGetWorkflowPayload(t, execCtx.Payloads[0])

		assert.Equal(t, "TEST-1", output.IssueKey)
		assert.Equal(t, "Task", output.IssueType)
		assert.Equal(t, "TEST", output.ProjectKey)
		assert.Equal(t, "In Progress", output.CurrentStatus)
		assert.Equal(t, "10002", output.CurrentStatusID)
		assert.Equal(t, "101010", output.WorkflowSchemeID)
		assert.Equal(t, "Default scheme", output.WorkflowSchemeName)
		assert.Equal(t, "task-workflow", output.WorkflowName)

		require.Len(t, output.Statuses, 3)
		statusCategories := map[string]string{}
		var foundCurrent bool
		for _, s := range output.Statuses {
			statusCategories[s.Name] = s.Category
			if s.Name == "In Progress" {
				assert.True(t, s.IsCurrent, "current status should be flagged")
				foundCurrent = true
			} else {
				assert.False(t, s.IsCurrent)
			}
		}
		assert.Equal(t, "TODO", statusCategories["To Do"])
		assert.Equal(t, "IN_PROGRESS", statusCategories["In Progress"])
		assert.Equal(t, "DONE", statusCategories["Done"])
		assert.True(t, foundCurrent)

		require.Len(t, output.AvailableTransitions, 2)
		assert.Equal(t, "Resolve", output.AvailableTransitions[0].Name)
		assert.Equal(t, "Done", output.AvailableTransitions[0].ToStatus)

		// transitions endpoint must request fields so the resolution-check works downstream.
		assert.Contains(t, httpContext.Requests[1].URL.String(), "expand=transitions.fields")
	})

	t.Run("team-managed project (no scheme) -> still emits current status + transitions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				// /workflowscheme/project returns no values for team-managed projects.
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values":[]}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
			Logger:         newLogger(),
		})

		require.NoError(t, err)
		output := unwrapGetWorkflowPayload(t, execCtx.Payloads[0])
		assert.Equal(t, "In Progress", output.CurrentStatus)
		assert.Equal(t, "", output.WorkflowName)
		assert.Empty(t, output.Statuses)
		require.Len(t, output.AvailableTransitions, 2)
	})

	t.Run("workflow scheme fetch failure is surfaced as a hard error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessage":"boom"}`))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch workflow scheme")
	})

	t.Run("workflow status fetch failure is surfaced as a hard error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectSchemeResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(schemeDetailResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueTypesResponse))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessage":"boom"}`))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load statuses")
	})

	t.Run("project issue type fetch failure does not silently fall back to default workflow", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectSchemeResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(schemeDetailResponse))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"errorMessage":"boom"}`))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve workflow")
	})

	t.Run("workflow/search prefix match for a different workflow returns an error", func(t *testing.T) {
		// scheme routes Task to "task-workflow", but workflow/search returns
		// the older "task-workflow-old" only. We must not pretend its
		// statuses belong to "task-workflow".
		prefixMatchOnly := `{"values":[{"id":{"name":"task-workflow-old"},"statuses":[
			{"id":"99","name":"Stale","statusCategory":"TODO"}
		]}]}`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectSchemeResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(schemeDetailResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueTypesResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(prefixMatchOnly))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Logger:         newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), `workflow "task-workflow" not found`)
	})

	t.Run("falls back to default workflow when issue type is not in the scheme mappings", func(t *testing.T) {
		schemeWithoutMapping := `{"id":101010,"name":"Default scheme","defaultWorkflow":"jira-default","issueTypeMappings":{}}`
		defaultWorkflowStatuses := `{"values":[{"id":{"name":"jira-default"},"statuses":[
			{"id":"10001","name":"To Do","statusCategory":"TODO"},
			{"id":"10002","name":"In Progress","statusCategory":"IN_PROGRESS"},
			{"id":"10003","name":"Done","statusCategory":"DONE"}
		]}]}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(transitionsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectSchemeResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(schemeWithoutMapping))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(issueTypesResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(defaultWorkflowStatuses))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
			Logger:         newLogger(),
		})

		require.NoError(t, err)
		output := unwrapGetWorkflowPayload(t, execCtx.Payloads[0])
		assert.Equal(t, "jira-default", output.WorkflowName)
	})
}

// unwrapGetWorkflowPayload extracts the GetWorkflowOutput from the wrapped
// `{type, timestamp, data}` envelope that ExecutionStateContext.Emit produces.
func unwrapGetWorkflowPayload(t *testing.T, payload any) GetWorkflowOutput {
	t.Helper()
	wrapped, ok := payload.(map[string]any)
	require.True(t, ok, "expected wrapped payload map, got %T", payload)
	out, ok := wrapped["data"].(GetWorkflowOutput)
	require.True(t, ok, "expected data to be GetWorkflowOutput, got %T", wrapped["data"])
	return out
}
