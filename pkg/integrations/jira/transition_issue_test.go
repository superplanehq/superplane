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

func Test__TransitionIssue__Setup(t *testing.T) {
	component := TransitionIssue{}

	t.Run("missing issue key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"targetStatus": "Done"},
		})

		require.ErrorContains(t, err, "issueKey is required")
	})

	t.Run("missing target status -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issueKey": "TEST-1"},
		})

		require.ErrorContains(t, err, "targetStatus is required")
	})

	t.Run("valid setup stores project and status metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{ID: "10000", Key: "TEST", Name: "Test Project"}},
			}),
			Metadata: metadataCtx,
			Configuration: map[string]any{
				"project":      "TEST",
				"issueKey":     "TEST-1",
				"targetStatus": "Done",
			},
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Done", nodeMetadata.Status)
		require.NotNil(t, nodeMetadata.Project)
		assert.Equal(t, "TEST", nodeMetadata.Project.Key)
	})
}

func Test__TransitionIssue__Execute(t *testing.T) {
	component := TransitionIssue{}

	t.Run("transitions issue with comment and resolution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"31","name":"Resolve","to":{"id":"10003","name":"Done"},"fields":{"resolution":{"required":false}}}]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1","fields":{"summary":"Done issue","status":{"name":"Done"}}}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"issueKey":     "TEST-1",
				"targetStatus": "Done",
				"comment":      "Ship it",
				"resolution":   "Done",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, TransitionIssuePayloadType, execCtx.Type)

		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/rest/api/3/issue/TEST-1/transitions")

		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		fields := payload["fields"].(map[string]any)
		assert.Equal(t, "Done", fields["resolution"].(map[string]any)["name"])
		update := payload["update"].(map[string]any)
		assert.Contains(t, update, "comment")
	})

	t.Run("unreachable status returns available transitions", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"21","name":"Start","to":{"id":"10002","name":"In Progress"}}]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"issueKey":     "TEST-1",
				"targetStatus": "Done",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Done")
		assert.Contains(t, err.Error(), "In Progress")
	})
}
