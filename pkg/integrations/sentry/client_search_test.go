package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SearchUnresolvedIssues(t *testing.T) {
	t.Run("scopes a query to the project and encodes is:unresolved", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"123","shortId":"PAYMENTS-1","title":"Database timeout"}]`),
			},
		}

		issues, err := testIssueClient(httpCtx).SearchUnresolvedIssues("payments", "timeout", 5)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "123", issues[0].ID)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(
			t,
			"https://sentry.io/api/0/projects/example/payments/issues/?query=is%3Aunresolved+timeout&limit=5",
			httpCtx.Requests[0].URL.String(),
		)
	})

	t.Run("ListNewestUnresolvedIssues still searches with is:unresolved alone", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[]`),
			},
		}

		_, err := testIssueClient(httpCtx).ListNewestUnresolvedIssues("payments", 10)
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(
			t,
			"https://sentry.io/api/0/projects/example/payments/issues/?query=is%3Aunresolved&limit=10",
			httpCtx.Requests[0].URL.String(),
		)
	})

	t.Run("an empty project searches the organization", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[]`),
			},
		}

		_, err := testIssueClient(httpCtx).SearchUnresolvedIssues("", "timeout", 5)
		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(
			t,
			"https://sentry.io/api/0/organizations/example/issues/?query=is%3Aunresolved+timeout&limit=5",
			httpCtx.Requests[0].URL.String(),
		)
	})
}
