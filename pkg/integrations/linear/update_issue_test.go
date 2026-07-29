package linear

import (
	"testing"

	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssue__Setup(t *testing.T) {
	component := UpdateIssue{}

	t.Run("missing team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"issue": "ENG-1", "title": "New"},
		})

		require.ErrorContains(t, err, "team is required")
	})

	t.Run("missing issue -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "title": "New"},
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("no fields enabled -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "issue": "ENG-1"},
		})

		require.ErrorContains(t, err, "at least one field must be enabled")
	})

	t.Run("title enabled but blank -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "issue": "ENG-1", "title": "   "},
		})

		require.ErrorContains(t, err, "title cannot be empty")
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "other", "issue": "ENG-1", "title": "New"},
		})

		require.ErrorContains(t, err, "team other not found")
	})

	t.Run("valid setup stores the team", func(t *testing.T) {
		metadataContext := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      metadataContext,
			Configuration: map[string]any{"team": "t1", "issue": "ENG-1", "state": "s1"},
		})

		require.NoError(t, err)
		metadata, ok := metadataContext.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Team)
		assert.Equal(t, "ENG", metadata.Team.Key)
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	component := UpdateIssue{}

	t.Run("emits the updated issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueUpdate":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","title":"Updated","url":"https://linear.app/acme/issue/ENG-142"}}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: executionState,
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "title": "Updated"},
		})

		require.NoError(t, err)
		assert.Equal(t, IssuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		issue, ok := wrapped["data"].(*Issue)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", issue.Identifier)
		assert.Equal(t, "Updated", issue.Title)
	})

	t.Run("sends the id and the enabled fields", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueUpdate":{"success":true,"issue":{"id":"i1","identifier":"ENG-142"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"team":        "t1",
				"issue":       "  ENG-142  ",
				"title":       "Updated",
				"description": "New body",
				"state":       "s1",
				"assignee":    "u1",
				"priority":    "1",
				"labels":      []string{"l1", "l2"},
				"project":     "p1",
			},
		})

		require.NoError(t, err)

		variables := graphQLVariablesFromRequest(t, httpContext)
		assert.Equal(t, "ENG-142", variables["id"])

		input, ok := variables["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Updated", input["title"])
		assert.Equal(t, "New body", input["description"])
		assert.Equal(t, "s1", input["stateId"])
		assert.Equal(t, "u1", input["assigneeId"])
		assert.Equal(t, float64(1), input["priority"])
		assert.Equal(t, []any{"l1", "l2"}, input["labelIds"])
		assert.Equal(t, "p1", input["projectId"])
	})

	t.Run("omits fields that were not enabled", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueUpdate":{"success":true,"issue":{"id":"i1","identifier":"ENG-142"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "title": "Updated"},
		})

		require.NoError(t, err)

		input, ok := graphQLVariablesFromRequest(t, httpContext)["input"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, input, "title")
		assert.NotContains(t, input, "description")
		assert.NotContains(t, input, "stateId")
		assert.NotContains(t, input, "assigneeId")
		assert.NotContains(t, input, "priority")
		assert.NotContains(t, input, "labelIds")
		assert.NotContains(t, input, "projectId")
	})

	t.Run("empty assignee unassigns the issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueUpdate":{"success":true,"issue":{"id":"i1","identifier":"ENG-142"}}}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142", "assignee": ""},
		})

		require.NoError(t, err)

		input, ok := graphQLVariablesFromRequest(t, httpContext)["input"].(map[string]any)
		require.True(t, ok)
		require.Contains(t, input, "assigneeId")
		assert.Nil(t, input["assigneeId"])
	})

	t.Run("no fields enabled -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-142"},
		})

		require.ErrorContains(t, err, "at least one field must be enabled")
	})

	t.Run("API failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Issue not found"}]}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationWithTeam(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"team": "t1", "issue": "ENG-999", "title": "Updated"},
		})

		require.ErrorContains(t, err, "Issue not found")
	})
}

func Test__UpdateIssue__BuildInput(t *testing.T) {
	t.Run("title is trimmed", func(t *testing.T) {
		input, err := buildUpdateIssueInput(
			UpdateIssueSpec{Title: "  Boom  "},
			updateIssueToggles{Title: true},
		)
		require.NoError(t, err)
		assert.Equal(t, "Boom", input["title"])
	})

	t.Run("enabled but blank title -> error", func(t *testing.T) {
		_, err := buildUpdateIssueInput(
			UpdateIssueSpec{Title: "   "},
			updateIssueToggles{Title: true},
		)
		require.ErrorContains(t, err, "title cannot be empty")
	})

	t.Run("blank description clears it", func(t *testing.T) {
		input, err := buildUpdateIssueInput(
			UpdateIssueSpec{Description: "   "},
			updateIssueToggles{Description: true},
		)
		require.NoError(t, err)
		require.Contains(t, input, "description")
		assert.Equal(t, "", input["description"])
	})

	t.Run("enabled but blank status -> error", func(t *testing.T) {
		_, err := buildUpdateIssueInput(
			UpdateIssueSpec{State: "  "},
			updateIssueToggles{State: true},
		)
		require.ErrorContains(t, err, "status cannot be empty")
	})

	t.Run("priority zero is sent", func(t *testing.T) {
		input, err := buildUpdateIssueInput(
			UpdateIssueSpec{Priority: "0"},
			updateIssueToggles{Priority: true},
		)
		require.NoError(t, err)
		assert.Equal(t, 0, input["priority"])
	})

	t.Run("out-of-range priority -> error", func(t *testing.T) {
		_, err := buildUpdateIssueInput(
			UpdateIssueSpec{Priority: "9"},
			updateIssueToggles{Priority: true},
		)
		require.ErrorContains(t, err, "invalid priority")
	})

	t.Run("blank labels are dropped, empty set clears", func(t *testing.T) {
		input, err := buildUpdateIssueInput(
			UpdateIssueSpec{Labels: []string{"", "  ", "l1"}},
			updateIssueToggles{Labels: true},
		)
		require.NoError(t, err)
		assert.Equal(t, []string{"l1"}, input["labelIds"])

		cleared, err := buildUpdateIssueInput(
			UpdateIssueSpec{Labels: []string{"  "}},
			updateIssueToggles{Labels: true},
		)
		require.NoError(t, err)
		assert.Equal(t, []string{}, cleared["labelIds"])
	})

	t.Run("empty project removes the issue from its project", func(t *testing.T) {
		input, err := buildUpdateIssueInput(
			UpdateIssueSpec{Project: ""},
			updateIssueToggles{Project: true},
		)
		require.NoError(t, err)
		require.Contains(t, input, "projectId")
		assert.Nil(t, input["projectId"])
	})

	t.Run("only enabled fields are present", func(t *testing.T) {
		input, err := buildUpdateIssueInput(
			UpdateIssueSpec{Title: "Boom", Description: "body", State: "s1"},
			updateIssueToggles{Title: true},
		)
		require.NoError(t, err)
		assert.Contains(t, input, "title")
		assert.NotContains(t, input, "description")
		assert.NotContains(t, input, "stateId")
	})
}
