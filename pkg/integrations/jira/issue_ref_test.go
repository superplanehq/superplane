package jira

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IssueKeyFromEventData(t *testing.T) {
	t.Run("reads an issue key from a webhook envelope", func(t *testing.T) {
		issueKey, ok := IssueKeyFromEventData(map[string]any{
			"type": IssueEventPayloadType,
			"data": map[string]any{
				"action": "created",
				"url":    "https://acme.atlassian.net/browse/ENG-5",
				"issue":  map[string]any{"key": "ENG-5"},
			},
		})

		require.True(t, ok)
		assert.Equal(t, "ENG-5", issueKey)
	})

	t.Run("ignores a non-jira event", func(t *testing.T) {
		_, ok := IssueKeyFromEventData(map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"issue": map[string]any{"key": "ENG-5"},
			},
		})
		assert.False(t, ok)
	})

	t.Run("ignores a jira event without an issue key", func(t *testing.T) {
		_, ok := IssueKeyFromEventData(map[string]any{
			"type": IssueEventPayloadType,
			"data": map[string]any{"action": "created"},
		})
		assert.False(t, ok)
	})
}

func Test__IssueKeyFromURL(t *testing.T) {
	t.Run("reads a browse URL", func(t *testing.T) {
		issueKey, ok := IssueKeyFromURL("https://acme.atlassian.net/browse/ENG-5")
		require.True(t, ok)
		assert.Equal(t, "ENG-5", issueKey)
	})

	t.Run("accepts a trailing slash", func(t *testing.T) {
		issueKey, ok := IssueKeyFromURL("https://acme.atlassian.net/browse/ENG-5/")
		require.True(t, ok)
		assert.Equal(t, "ENG-5", issueKey)
	})

	t.Run("ignores a query string", func(t *testing.T) {
		issueKey, ok := IssueKeyFromURL("https://acme.atlassian.net/browse/ENG-5?focusedCommentId=1")
		require.True(t, ok)
		assert.Equal(t, "ENG-5", issueKey)
	})

	t.Run("ignores a non-browse URL", func(t *testing.T) {
		_, ok := IssueKeyFromURL("https://acme.atlassian.net/jira/software/projects/ENG/boards/1")
		assert.False(t, ok)
	})

	t.Run("does not treat ENG-5 as ENG-50", func(t *testing.T) {
		issueKey, ok := IssueKeyFromURL("https://acme.atlassian.net/browse/ENG-50")
		require.True(t, ok)
		assert.Equal(t, "ENG-50", issueKey)
		assert.NotEqual(t, "ENG-5", issueKey)

		five, ok := IssueKeyFromURL("https://acme.atlassian.net/browse/ENG-5")
		require.True(t, ok)
		assert.Equal(t, "ENG-5", five)
	})
}

func Test__IssueURLFragment(t *testing.T) {
	assert.Equal(t, "/browse/ENG-5", IssueURLFragment("ENG-5"))
	assert.Equal(t, "", IssueURLFragment("  "))
}
