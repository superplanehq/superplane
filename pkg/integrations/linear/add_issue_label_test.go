package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AddIssueLabel__Setup(t *testing.T) {
	component := AddIssueLabel{}

	t.Run("missing team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issue": "ENG-142", "labels": []string{"l1"}},
		})

		require.ErrorContains(t, err, "team is required")
	})

	t.Run("missing issue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "labels": []string{"l1"}},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("no labels of either kind -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"  "}, "newLabels": []string{"  "}},
		})

		require.ErrorContains(t, err, "at least one label is required")
	})

	t.Run("new labels only is valid", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "issue": "ENG-142", "newLabels": []string{"needs-triage"}},
		})

		require.NoError(t, err)
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "other", "issue": "ENG-142", "labels": []string{"l1"}},
		})

		require.ErrorContains(t, err, "team other not found")
	})

	t.Run("valid setup stores the team", func(t *testing.T) {
		metadataContext := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      metadataContext,
			Configuration: map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"l1"}},
		})

		require.NoError(t, err)
		metadata, ok := metadataContext.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Team)
		assert.Equal(t, "ENG", metadata.Team.Key)
	})
}

func Test__AddIssueLabel__Execute(t *testing.T) {
	component := AddIssueLabel{}

	t.Run("adds every label and emits the updated issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueAddLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","labels":{"nodes":[{"id":"l1","name":"bug"}]}}}}}`),
				jsonResponse(`{"data":{"issueAddLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","labels":{"nodes":[{"id":"l1","name":"bug"},{"id":"l2","name":"urgent"}]}}}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"l1", "l2"}},
		})

		require.NoError(t, err)
		assert.Equal(t, IssuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		issue, ok := wrapped["data"].(*Issue)
		require.True(t, ok)

		// The emitted issue reflects the state after the last addition.
		require.Len(t, issue.Labels, 2)

		// One request per label, each carrying its own labelId.
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "ENG-142", variablesFromRequest(t, httpContext, 0)["id"])
		assert.Equal(t, "l1", variablesFromRequest(t, httpContext, 0)["labelId"])
		assert.Equal(t, "l2", variablesFromRequest(t, httpContext, 1)["labelId"])
	})

	t.Run("blank labels are skipped", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueAddLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"", "  ", "l1"}},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "l1", variablesFromRequest(t, httpContext, 0)["labelId"])
	})

	t.Run("creates a missing new label then adds it", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueLabels":{"nodes":[{"id":"l1","name":"bug"}],"pageInfo":{"hasNextPage":false}}}}`),
				jsonResponse(`{"data":{"issueLabelCreate":{"success":true,"issueLabel":{"id":"lnew","name":"needs-triage"}}}}`),
				jsonResponse(`{"data":{"issueAddLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","labels":{"nodes":[{"id":"lnew","name":"needs-triage"}]}}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "newLabels": []string{"needs-triage"}},
		})

		require.NoError(t, err)

		// List existing labels, create the missing one, then attach it.
		require.Len(t, httpContext.Requests, 3)
		createInput, ok := variablesFromRequest(t, httpContext, 1)["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "needs-triage", createInput["name"])
		assert.Equal(t, "t1", createInput["teamId"])
		assert.Equal(t, "lnew", variablesFromRequest(t, httpContext, 2)["labelId"])
	})

	t.Run("reuses an existing label matching a new-label name", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueLabels":{"nodes":[{"id":"l1","name":"bug"}],"pageInfo":{"hasNextPage":false}}}}`),
				jsonResponse(`{"data":{"issueAddLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","labels":{"nodes":[{"id":"l1","name":"bug"}]}}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "newLabels": []string{"Bug"}},
		})

		require.NoError(t, err)

		// No label is created; the existing "bug" is reused (case-insensitive).
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "l1", variablesFromRequest(t, httpContext, 1)["labelId"])
	})

	t.Run("dedupes a label picked and also typed as new", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueLabels":{"nodes":[{"id":"l1","name":"bug"}],"pageInfo":{"hasNextPage":false}}}}`),
				jsonResponse(`{"data":{"issueAddLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","labels":{"nodes":[{"id":"l1","name":"bug"}]}}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"l1"}, "newLabels": []string{"bug"}},
		})

		require.NoError(t, err)

		// ListLabels resolves "bug" to l1, which is already picked, so it is
		// attached once: one list call + one add call, no duplicate add.
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "l1", variablesFromRequest(t, httpContext, 1)["labelId"])
	})

	t.Run("missing issue -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "labels": []string{"l1"}},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("API failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Entity not found"}]}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"l1"}},
		})

		require.ErrorContains(t, err, "Entity not found")
	})
}
