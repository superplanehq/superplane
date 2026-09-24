package jira

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__parseIssueWebhookConflictURL(t *testing.T) {
	t.Run("extracts the blocking callback from Atlassian's error", func(t *testing.T) {
		blocker := "https://app.superplane.com/api/v1/webhooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6"
		err := fmt.Errorf("failed to create webhook: Only a single URL per user is allowed to be registered via REST API. The currently used URL: %s", blocker)

		got, ok := parseIssueWebhookConflictURL(err)
		require.True(t, ok)
		assert.Equal(t, blocker, got)
	})

	t.Run("ignores JQL and other create errors", func(t *testing.T) {
		_, ok := parseIssueWebhookConflictURL(errors.New("failed to create webhook: The clause myClause is unsupported"))
		assert.False(t, ok)
	})
}

func Test__sameOriginCallbacks(t *testing.T) {
	requested := "https://app.superplane.com/api/v1/webhooks/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

	assert.True(t, sameOriginCallbacks(requested, "https://app.superplane.com/api/v1/webhooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6"))
	assert.False(t, sameOriginCallbacks(requested, "https://other.example.com/api/v1/webhooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6"))
}

func Test__superplaneWebhookIDFromCallbackURL(t *testing.T) {
	id, err := superplaneWebhookIDFromCallbackURL("https://app.superplane.com/api/v1/webhooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6")
	require.NoError(t, err)
	assert.Equal(t, "ed13e750-bd47-4ae4-9400-4d75ef42eab6", id)

	_, err = superplaneWebhookIDFromCallbackURL("https://app.superplane.com/hooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6")
	require.Error(t, err)
}

func Test__issueWebhookIDsMatchingURL(t *testing.T) {
	blocker := "https://app.superplane.com/api/v1/webhooks/ed13e750-bd47-4ae4-9400-4d75ef42eab6"
	webhooks := []IssueWebhook{
		{ID: 1, URL: blocker},
		{ID: 2, URL: "https://app.superplane.com/api/v1/webhooks/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
		{ID: 3, URL: ""},
	}

	assert.Equal(t, []int64{1}, issueWebhookIDsMatchingURL(webhooks, blocker))
	assert.Empty(t, issueWebhookIDsMatchingURL(webhooks, "https://app.superplane.com/api/v1/webhooks/cccccccc-cccc-4ccc-8ccc-cccccccccccc"))
}
