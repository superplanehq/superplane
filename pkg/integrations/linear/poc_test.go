package linear

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIssueCreatedWebhook(t *testing.T) {
	t.Run("returns normalized issue.created event", func(t *testing.T) {
		body := []byte(`{
			"action": "create",
			"type": "Issue",
			"data": {
				"id": "issue_123",
				"identifier": "ENG-42",
				"title": "Investigate failed deploy webhook",
				"url": "https://linear.app/acme/issue/ENG-42",
				"team": {"id":"team_1","name":"Engineering"},
				"labels": [{"id":"label_1","name":"bug"},{"id":"label_2","name":"prod"}]
			}
		}`)

		got, err := ParseIssueCreatedWebhook(body)
		require.NoError(t, err)
		assert.Equal(t, "issue_123", got.IssueID)
		assert.Equal(t, "ENG-42", got.Identifier)
		assert.Equal(t, "team_1", got.TeamID)
		assert.Equal(t, []string{"bug", "prod"}, got.IssueLabels)
	})

	t.Run("trims and deduplicates labels", func(t *testing.T) {
		body := []byte(`{
			"action": "create",
			"type": "Issue",
			"data": {
				"id": "issue_124",
				"identifier": "ENG-43",
				"title": "Label normalization",
				"url": "https://linear.app/acme/issue/ENG-43",
				"team": {"id":"team_1","name":"Engineering"},
				"labels": [
					{"id":"label_1","name":" bug "},
					{"id":"label_2","name":"bug"},
					{"id":"label_3","name":""},
					{"id":"label_4","name":"  prod  "}
				]
			}
		}`)

		got, err := ParseIssueCreatedWebhook(body)
		require.NoError(t, err)
		assert.Equal(t, []string{"bug", "prod"}, got.IssueLabels)
	})

	t.Run("filters out non issue.create events", func(t *testing.T) {
		body := []byte(`{
			"action": "update",
			"type": "Issue",
			"data": {"id":"issue_123","team":{"id":"team_1"}}
		}`)

		_, err := ParseIssueCreatedWebhook(body)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotIssueCreatedEvent))
	})

	t.Run("fails when required fields are missing", func(t *testing.T) {
		body := []byte(`{
			"action": "create",
			"type": "Issue",
			"data": {"id":"","team":{"id":""}}
		}`)

		_, err := ParseIssueCreatedWebhook(body)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidWebhookPayload))
	})
}

func TestBuildIssueCreateVariables(t *testing.T) {
	t.Run("builds input variables with optional fields", func(t *testing.T) {
		vars, err := BuildIssueCreateVariables(CreateIssueInput{
			TeamID:      "team_1",
			Title:       "Auto-created issue from SuperPlane",
			Description: "Generated from PoC",
			AssigneeID:  "user_1",
			LabelIDs:    []string{"label_bug", "label_backend"},
			Priority:    2,
			StateID:     "state_todo",
		})
		require.NoError(t, err)

		input := vars["input"].(map[string]any)
		assert.Equal(t, "team_1", input["teamId"])
		assert.Equal(t, 2, input["priority"])
		assert.Equal(t, "state_todo", input["stateId"])
		assert.Equal(t, []string{"label_bug", "label_backend"}, input["labelIds"])
	})

	t.Run("rejects invalid priority", func(t *testing.T) {
		_, err := BuildIssueCreateVariables(CreateIssueInput{
			TeamID:   "team_1",
			Title:    "x",
			Priority: 9,
		})
		require.Error(t, err)
		assert.Equal(t, "priority must be in range 0..4", err.Error())
	})

	t.Run("trims required and optional fields", func(t *testing.T) {
		vars, err := BuildIssueCreateVariables(CreateIssueInput{
			TeamID:      " team_1 ",
			Title:       "  Auto-created issue from SuperPlane  ",
			Description: "  Generated from PoC  ",
			AssigneeID:  " user_1 ",
			LabelIDs:    []string{" label_bug ", "label_bug", "", " label_backend "},
			Priority:    1,
			StateID:     " state_todo ",
		})
		require.NoError(t, err)

		input := vars["input"].(map[string]any)
		assert.Equal(t, "team_1", input["teamId"])
		assert.Equal(t, "Auto-created issue from SuperPlane", input["title"])
		assert.Equal(t, "Generated from PoC", input["description"])
		assert.Equal(t, "user_1", input["assigneeId"])
		assert.Equal(t, []string{"label_bug", "label_backend"}, input["labelIds"])
		assert.Equal(t, "state_todo", input["stateId"])
	})
}
