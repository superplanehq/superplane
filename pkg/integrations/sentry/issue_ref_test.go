package sentry

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__IssueIDFromEventData(t *testing.T) {
	t.Run("reads a string issue id from a webhook envelope", func(t *testing.T) {
		issueID, ok := IssueIDFromEventData(map[string]any{
			"type": IssuePayloadType,
			"data": map[string]any{
				"resource": "issue",
				"action":   "created",
				"data": map[string]any{
					"issue": map[string]any{"id": "7670162495", "title": "boom"},
				},
			},
		})

		require.True(t, ok)
		assert.Equal(t, "7670162495", issueID)
	})

	t.Run("reads a numeric issue id preserved as json.Number", func(t *testing.T) {
		issueID, ok := IssueIDFromEventData(map[string]any{
			"type": IssuePayloadType,
			"data": map[string]any{
				"data": map[string]any{
					"issue": map[string]any{"id": json.Number("123")},
				},
			},
		})

		require.True(t, ok)
		assert.Equal(t, "123", issueID)
	})

	t.Run("ignores a non-sentry event", func(t *testing.T) {
		_, ok := IssueIDFromEventData(map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"data": map[string]any{
					"issue": map[string]any{"id": "12"},
				},
			},
		})
		assert.False(t, ok)
	})

	t.Run("ignores a sentry event without an issue id", func(t *testing.T) {
		_, ok := IssueIDFromEventData(map[string]any{
			"type": IssuePayloadType,
			"data": map[string]any{"action": "created"},
		})
		assert.False(t, ok)
	})
}

func Test__IssueIDFromURL(t *testing.T) {
	t.Run("reads an org-host issue URL", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://acme.sentry.io/issues/12/")
		require.True(t, ok)
		assert.Equal(t, "12", issueID)
	})

	t.Run("accepts a missing trailing slash", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://acme.sentry.io/issues/12")
		require.True(t, ok)
		assert.Equal(t, "12", issueID)
	})

	t.Run("ignores a query string", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://acme.sentry.io/issues/12/?project=1&query=is%3Aunresolved")
		require.True(t, ok)
		assert.Equal(t, "12", issueID)
	})

	t.Run("reads an organizations issue URL", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://sentry.io/organizations/acme/issues/12/")
		require.True(t, ok)
		assert.Equal(t, "12", issueID)
	})

	t.Run("ignores a non-issue URL", func(t *testing.T) {
		_, ok := IssueIDFromURL("https://acme.sentry.io/projects/frontend/")
		assert.False(t, ok)
	})

	t.Run("ignores a GitHub issue URL", func(t *testing.T) {
		_, ok := IssueIDFromURL("https://github.com/acme/payments/issues/12")
		assert.False(t, ok)
	})

	t.Run("ignores a GitHub-style path on another host", func(t *testing.T) {
		_, ok := IssueIDFromURL("https://github.internal.example/acme/payments/issues/12")
		assert.False(t, ok)
	})

	t.Run("reads a self-hosted issue URL", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://sentry.internal.example/issues/12/")
		require.True(t, ok)
		assert.Equal(t, "12", issueID)
	})

	t.Run("reads a self-hosted organizations issue URL", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://sentry.internal.example/organizations/acme/issues/12/")
		require.True(t, ok)
		assert.Equal(t, "12", issueID)
	})

	t.Run("does not treat 12 as 123", func(t *testing.T) {
		issueID, ok := IssueIDFromURL("https://acme.sentry.io/issues/123/")
		require.True(t, ok)
		assert.Equal(t, "123", issueID)
		assert.NotEqual(t, "12", issueID)

		twelve, ok := IssueIDFromURL("https://acme.sentry.io/issues/12/")
		require.True(t, ok)
		assert.Equal(t, "12", twelve)
	})
}

func Test__IssueURLFragment(t *testing.T) {
	assert.Equal(t, "/issues/12", IssueURLFragment("12"))
	assert.Equal(t, "", IssueURLFragment("  "))
}

func Test__IssueStatusIsSettled(t *testing.T) {
	assert.True(t, IssueStatusIsSettled(IssueStatusResolved))
	assert.True(t, IssueStatusIsSettled(IssueStatusResolvedInNextRelease))
	assert.True(t, IssueStatusIsSettled(IssueStatusIgnored))
	assert.False(t, IssueStatusIsSettled(IssueStatusUnresolved))
	assert.False(t, IssueStatusIsSettled(""))
}

func Test__IsRetryableAPIError(t *testing.T) {
	t.Run("retries rate limits, timeouts, and server errors", func(t *testing.T) {
		assert.True(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusTooManyRequests}))
		assert.True(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusRequestTimeout}))
		assert.True(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusInternalServerError}))
		assert.True(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusBadGateway}))
	})

	t.Run("does not retry client errors", func(t *testing.T) {
		assert.False(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusBadRequest}))
		assert.False(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusUnauthorized}))
		assert.False(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusForbidden}))
		assert.False(t, IsRetryableAPIError(&apiError{StatusCode: http.StatusNotFound}))
	})

	t.Run("retries unknown transport failures", func(t *testing.T) {
		assert.True(t, IsRetryableAPIError(errors.New("connection reset")))
	})

	t.Run("treats a nil error as settled", func(t *testing.T) {
		assert.False(t, IsRetryableAPIError(nil))
	})
}
