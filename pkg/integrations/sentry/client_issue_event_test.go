package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetIssueEvent(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{
				"eventID":"evt-latest",
				"title":"TypeError: boom",
				"culprit":"app.js",
				"entries":[{"type":"exception","data":{"values":[{"type":"TypeError","value":"boom","stacktrace":{"frames":[{"filename":"app.js","function":"run","lineNo":12,"inApp":true}]}}]}}]
			}`),
		},
	}

	event, err := testIssueClient(httpCtx).GetIssueEvent("123", IssueEventLatest)
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, "evt-latest", event.EventID)
	assert.True(t, event.HasStack())
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(
		t,
		"https://sentry.io/api/0/organizations/example/issues/123/events/latest/",
		httpCtx.Requests[0].URL.String(),
	)
}

func Test__GetPreferredIssueEvent(t *testing.T) {
	t.Run("keeps latest when it has a stack", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{
					"eventID":"latest",
					"entries":[{"type":"exception","data":{"values":[{"stacktrace":{"frames":[{"filename":"a.go","function":"main","lineNo":1}]}}]}}]
				}`),
			},
		}

		event, err := testIssueClient(httpCtx).GetPreferredIssueEvent("123")
		require.NoError(t, err)
		assert.Equal(t, "latest", event.EventID)
		assert.Len(t, httpCtx.Requests, 1)
	})

	t.Run("uses recommended when latest has no stack", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"eventID":"latest","entries":[{"type":"message","data":{"formatted":"json event"}}]}`),
				sentryMockResponse(http.StatusOK, `{
					"eventID":"recommended",
					"entries":[{"type":"exception","data":{"values":[{"stacktrace":{"frames":[{"filename":"app.js","function":"fail","lineNo":9}]}}]}}]
				}`),
			},
		}

		event, err := testIssueClient(httpCtx).GetPreferredIssueEvent("123")
		require.NoError(t, err)
		assert.Equal(t, "recommended", event.EventID)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(
			t,
			"https://sentry.io/api/0/organizations/example/issues/123/events/recommended/",
			httpCtx.Requests[1].URL.String(),
		)
	})

	t.Run("keeps latest when recommended fetch fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"eventID":"latest","entries":[]}`),
				sentryMockResponse(http.StatusNotFound, `{"detail":"not found"}`),
			},
		}

		event, err := testIssueClient(httpCtx).GetPreferredIssueEvent("123")
		require.NoError(t, err)
		assert.Equal(t, "latest", event.EventID)
	})
}

func testIssueClient(httpCtx *contexts.HTTPContext) *Client {
	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":   "https://sentry.io",
			"userToken": "user-token",
		},
		Metadata: Metadata{
			Organization: &OrganizationSummary{Slug: "example"},
		},
	})
	if err != nil {
		panic(err)
	}
	return client
}
