package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func integrationWithTeam() *contexts.IntegrationContext {
	return newAuthorizedIntegrationWithMetadata(Metadata{
		Teams: []Team{{ID: "t1", Key: "ENG", Name: "Engineering"}},
	})
}

func Test__CreateIssue__Setup(t *testing.T) {
	component := CreateIssue{}

	t.Run("missing team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"title": "Boom"},
		})

		require.ErrorContains(t, err, "team is required")
	})

	t.Run("missing title -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1"},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("blank title -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "title": "   "},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "other", "title": "Boom"},
		})

		require.ErrorContains(t, err, "team other not found")
	})

	t.Run("valid setup stores the team", func(t *testing.T) {
		metadataContext := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      metadataContext,
			Configuration: map[string]any{"team": "t1", "title": "Boom"},
		})

		require.NoError(t, err)
		metadata, ok := metadataContext.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Team)
		assert.Equal(t, "ENG", metadata.Team.Key)
	})
}

func Test__CreateIssue__Execute(t *testing.T) {
	component := CreateIssue{}

	t.Run("emits the created issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueCreate":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","title":"Boom","url":"https://linear.app/acme/issue/ENG-142"}}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"team": "t1", "title": "Boom"},
		})

		require.NoError(t, err)
		assert.Equal(t, IssuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		issue, ok := wrapped["data"].(*Issue)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", issue.Identifier)
		assert.Equal(t, "https://linear.app/acme/issue/ENG-142", issue.URL)
	})

	t.Run("sends the optional fields it was given", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueCreate":{"success":true,"issue":{"id":"i1","identifier":"ENG-1"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"team":        "t1",
				"title":       "Boom",
				"description": "It broke",
				"state":       "s1",
				"assignee":    "u1",
				"priority":    "2",
				"labels":      []string{"l1", "l2"},
			},
		})

		require.NoError(t, err)

		input := createIssueInputFromRequest(t, httpContext)
		assert.Equal(t, "t1", input["teamId"])
		assert.Equal(t, "It broke", input["description"])
		assert.Equal(t, "s1", input["stateId"])
		assert.Equal(t, "u1", input["assigneeId"])
		assert.Equal(t, float64(2), input["priority"])
		assert.Equal(t, []any{"l1", "l2"}, input["labelIds"])
	})

	t.Run("omits optional fields that were left empty", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueCreate":{"success":true,"issue":{"id":"i1","identifier":"ENG-1"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "title": "Boom"},
		})

		require.NoError(t, err)

		input := createIssueInputFromRequest(t, httpContext)
		assert.NotContains(t, input, "stateId")
		assert.NotContains(t, input, "assigneeId")
		assert.NotContains(t, input, "priority")
		assert.NotContains(t, input, "labelIds")
		assert.NotContains(t, input, "description")
	})

	t.Run("API failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Team not found"}]}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "title": "Boom"},
		})

		require.ErrorContains(t, err, "Team not found")
	})
}

func Test__CreateIssue__BuildInput(t *testing.T) {
	t.Run("priority zero is sent", func(t *testing.T) {
		input, err := buildCreateIssueInput(CreateIssueSpec{Team: "t1", Title: "Boom", Priority: "0"})
		require.NoError(t, err)
		assert.Equal(t, 0, input["priority"])
	})

	t.Run("non-numeric priority -> error", func(t *testing.T) {
		_, err := buildCreateIssueInput(CreateIssueSpec{Team: "t1", Title: "Boom", Priority: "urgent"})
		require.ErrorContains(t, err, "invalid priority")
	})

	t.Run("out-of-range priority -> error", func(t *testing.T) {
		_, err := buildCreateIssueInput(CreateIssueSpec{Team: "t1", Title: "Boom", Priority: "9"})
		require.ErrorContains(t, err, "invalid priority")
	})

	t.Run("blank labels are dropped", func(t *testing.T) {
		input, err := buildCreateIssueInput(CreateIssueSpec{Team: "t1", Title: "Boom", Labels: []string{"", "  ", "l1"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"l1"}, input["labelIds"])
	})

	t.Run("only blank labels means no labelIds", func(t *testing.T) {
		input, err := buildCreateIssueInput(CreateIssueSpec{Team: "t1", Title: "Boom", Labels: []string{"", "  "}})
		require.NoError(t, err)
		assert.NotContains(t, input, "labelIds")
	})

	t.Run("title is trimmed", func(t *testing.T) {
		input, err := buildCreateIssueInput(CreateIssueSpec{Team: "t1", Title: "  Boom  "})
		require.NoError(t, err)
		assert.Equal(t, "Boom", input["title"])
	})
}

func createIssueInputFromRequest(t *testing.T, httpContext *contexts.HTTPContext) map[string]any {
	t.Helper()

	require.Len(t, httpContext.Requests, 1)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	payload := struct {
		Variables struct {
			Input map[string]any `json:"input"`
		} `json:"variables"`
	}{}

	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Variables.Input
}
