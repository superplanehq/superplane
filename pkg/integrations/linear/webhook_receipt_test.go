package linear

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeWebhook(t *testing.T) {
	t.Run("reads an issue event without the description or actor", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(EventHeader, IssueResourceType)
		body := []byte(`{
			"action": "create",
			"type": "Issue",
			"actor": {"email": "ada@example.com"},
			"url": "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails",
			"data": {
				"id": "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9",
				"identifier": "ENG-142",
				"title": "Deploy pipeline fails",
				"description": "secret-description",
				"team": {"key": "ENG", "name": "Engineering"}
			}
		}`)

		summary := SummarizeWebhook(headers, body)

		assert.Equal(t, IssueResourceType, summary.EventType)
		assert.Equal(t, "create", summary.Action)
		assert.Equal(t, "ENG-142", summary.IssueIdentifier)
		assert.Equal(t, "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9", summary.IssueID)
		assert.Equal(t, "ENG", summary.TeamKey)
		assert.Equal(t, "acme", summary.WorkspaceKey)
		assert.NotContains(t, summary.IssueIdentifier+summary.IssueID+summary.TeamKey+summary.WorkspaceKey, "secret-description")
		assert.NotContains(t, summary.IssueIdentifier+summary.IssueID, "ada@example.com")
	})

	t.Run("reads the parent issue from a comment event", func(t *testing.T) {
		body := []byte(`{
			"action": "create",
			"type": "Comment",
			"url": "https://linear.app/acme/issue/ENG-142/deploy#comment-1",
			"data": {
				"id": "comment-1",
				"body": "secret-comment",
				"issueId": "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9",
				"issue": {
					"id": "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9",
					"identifier": "ENG-142",
					"team": {"key": "ENG"}
				}
			}
		}`)

		summary := SummarizeWebhook(nil, body)

		assert.Equal(t, CommentResourceType, summary.EventType)
		assert.Equal(t, "ENG-142", summary.IssueIdentifier)
		assert.Equal(t, "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9", summary.IssueID)
		assert.Equal(t, "ENG", summary.TeamKey)
		assert.NotEqual(t, "comment-1", summary.IssueID)
		assert.NotContains(t, summary.IssueIdentifier+summary.IssueID, "secret-comment")
	})
}

func TestReceiptIDFromEventData(t *testing.T) {
	receiptID := uuid.New()
	found, ok := ReceiptIDFromEventData(map[string]any{
		"type": IssuePayloadType,
		"data": map[string]any{
			ReceiptField: receiptID.String(),
		},
	})
	require.True(t, ok)
	assert.Equal(t, receiptID, found)

	_, ok = ReceiptIDFromEventData(map[string]any{
		"type": "sentry.issue",
		"data": map[string]any{
			ReceiptField: receiptID.String(),
		},
	})
	assert.False(t, ok)
}
