package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__issueQueryResult(t *testing.T) {
	const response = `{
	  "repository": {
	    "issues": {
	      "nodes": [
	        {
	          "id": "I_kwDOABCD",
	          "databaseId": 918273,
	          "number": 42,
	          "title": "Handle duplicate refunds on retry",
	          "body": "A retried refund charges the customer twice.",
	          "url": "https://github.com/acme/backlog/issues/42",
	          "state": "OPEN",
	          "createdAt": "2026-08-01T10:00:00Z",
	          "updatedAt": "2026-08-02T11:30:00Z",
	          "author": { "login": "ana" },
	          "labels": { "nodes": [{ "name": "bug", "description": "Broken behavior", "color": "d73a4a" }] },
	          "assignees": { "nodes": [{ "login": "bruno", "databaseId": 77 }] },
	          "comments": { "totalCount": 3 }
	        },
	        {
	          "id": "I_kwDOEFGH",
	          "databaseId": 918274,
	          "number": 41,
	          "title": "Upgrade the Node 20 base image",
	          "body": "",
	          "url": "https://github.com/acme/backlog/issues/41",
	          "state": "OPEN",
	          "createdAt": "2026-07-30T09:00:00Z",
	          "updatedAt": "2026-07-30T09:00:00Z",
	          "author": null,
	          "labels": { "nodes": [] },
	          "assignees": { "nodes": [] },
	          "comments": { "totalCount": 0 }
	        }
	      ]
	    }
	  }
	}`

	var result issueQueryResult
	require.NoError(t, json.Unmarshal([]byte(response), &result))
	require.Len(t, result.Repository.Issues.Nodes, 2)

	t.Run("an issue reads like its REST body", func(t *testing.T) {
		issue := result.Repository.Issues.Nodes[0].issue()

		assert.Equal(t, int64(918273), issue.GetID())
		assert.Equal(t, "I_kwDOABCD", issue.GetNodeID())
		assert.Equal(t, 42, issue.GetNumber())
		assert.Equal(t, "Handle duplicate refunds on retry", issue.GetTitle())
		assert.Equal(t, "A retried refund charges the customer twice.", issue.GetBody())
		assert.Equal(t, "https://github.com/acme/backlog/issues/42", issue.GetHTMLURL())
		assert.Equal(t, "open", issue.GetState())
		assert.Equal(t, 3, issue.GetComments())
		assert.Equal(t, "2026-08-01T10:00:00Z", issue.GetCreatedAt().Format("2006-01-02T15:04:05Z"))
		assert.Equal(t, "ana", issue.GetUser().GetLogin())

		require.Len(t, issue.Labels, 1)
		assert.Equal(t, "bug", issue.Labels[0].GetName())

		require.Len(t, issue.Assignees, 1)
		assert.Equal(t, "bruno", issue.Assignees[0].GetLogin())
		assert.Equal(t, int64(77), issue.Assignees[0].GetID())
	})

	t.Run("an issue without an author or lists stays readable", func(t *testing.T) {
		issue := result.Repository.Issues.Nodes[1].issue()

		assert.Equal(t, 41, issue.GetNumber())
		assert.Nil(t, issue.User)
		assert.Empty(t, issue.Labels)
		assert.Empty(t, issue.Assignees)
	})
}
